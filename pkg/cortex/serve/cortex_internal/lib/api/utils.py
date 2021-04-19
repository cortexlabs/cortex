import os
import datetime
import glob
import shutil
import json
from typing import Optional, Tuple

from cortex_internal.lib import util
from cortex_internal.lib.storage import S3
from cortex_internal.lib.type import PredictorType
from cortex_internal.lib.exceptions import CortexException
from cortex_internal.lib.model import validate_model_paths
from cortex_internal.lib.log import configure_logger

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])

def get_spec(
    spec_path: str,
    cache_dir: str,
    region: str,
    spec_name: str = "api_spec.json",
) -> Tuple[S3, dict]:
    """
    Args:
        spec_path: Path to API spec (i.e. "s3://cortex-dev-0/apis/iris-classifier/api/69b93378fa5c0218-jy1fjtyihu-9fcc10739e7fc8050cefa8ca27ece1ee/master-spec.json").
        cache_dir: Local directory where the API spec gets saved to.
        region: Region of the bucket.
        spec_name: File name of the spec as it is saved on disk.
    """

    bucket, key = S3.deconstruct_s3_path(spec_path)
    storage = S3(bucket=bucket, region=region)

    local_spec_path = os.path.join(cache_dir, spec_name)
    if not os.path.isfile(local_spec_path):
        storage.download_file(key, local_spec_path)

    return storage, read_json(local_spec_path)


def read_json(json_path: str):
    with open(json_path) as json_file:
        return json.load(json_file)

def model_downloader(
    predictor_type: PredictorType,
    bucket_name: str,
    model_name: str,
    model_version: str,
    model_path: str,
    temp_dir: str,
    model_dir: str,
) -> Optional[datetime.datetime]:
    """
    Downloads model to disk. Validates the s3 model path and the downloaded model.

    Args:
        predictor_type: The predictor type as implemented by the API.
        bucket_name: Name of the bucket where the model is stored.
        model_name: Name of the model. Is part of the model's local path.
        model_version: Version of the model. Is part of the model's local path.
        model_path: Model prefix of the versioned model.
        temp_dir: Where to temporarily store the model for validation.
        model_dir: The top directory of where all models are stored locally.

    Returns:
        The model's timestamp. None if the model didn't pass the validation, if it doesn't exist or if there are not enough permissions.
    """

    logger.info(
        f"downloading from bucket {bucket_name}/{model_path}, model {model_name} of version {model_version}, temporarily to {temp_dir} and then finally to {model_dir}"
    )

    client = S3(bucket_name)

    # validate upstream S3 model
    sub_paths, ts = client.search(model_path)
    try:
        validate_model_paths(sub_paths, predictor_type, model_path)
    except CortexException:
        logger.info(f"failed validating model {model_name} of version {model_version}")
        return None

    # download model to temp dir
    temp_dest = os.path.join(temp_dir, model_name, model_version)
    try:
        client.download_dir_contents(model_path, temp_dest)
    except CortexException:
        logger.info(
            f"failed downloading model {model_name} of version {model_version} to temp dir {temp_dest}"
        )
        shutil.rmtree(temp_dest)
        return None

    # validate model
    model_contents = glob.glob(os.path.join(temp_dest, "**"), recursive=True)
    model_contents = util.remove_non_empty_directory_paths(model_contents)
    try:
        validate_model_paths(model_contents, predictor_type, temp_dest)
    except CortexException:
        logger.info(
            f"failed validating model {model_name} of version {model_version} from temp dir"
        )
        shutil.rmtree(temp_dest)
        return None

    # move model to dest dir
    model_top_dir = os.path.join(model_dir, model_name)
    ondisk_model_version = os.path.join(model_top_dir, model_version)
    logger.info(
        f"moving model {model_name} of version {model_version} to final dir {ondisk_model_version}"
    )
    if os.path.isdir(ondisk_model_version):
        shutil.rmtree(ondisk_model_version)
    shutil.move(temp_dest, ondisk_model_version)

    return max(ts)
