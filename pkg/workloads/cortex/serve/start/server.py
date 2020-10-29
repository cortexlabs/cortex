import sys

import uvicorn
import yaml

def main():
    uds = sys.argv[1]

    with open("/src/cortex/serve/log_config.yaml", "r") as f:
        log_config = yaml.load(f, yaml.FullLoader)

    uvicorn.run(
        "cortex.serve.wsgi:app",
        uds=uds,
        log_config=log_config,
        log_level="info",
    )

if __name__ == "__main__":
    main()