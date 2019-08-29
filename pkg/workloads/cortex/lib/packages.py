from subprocess import run

from cortex.lib.exceptions import UserException, CortexException


def install(project_path):
    completed_process = run("pip3 install -r {}/requirements.txt".format(project_path).split())
    if completed_process.returncode != 0:
        raise UserException("installing packages")
