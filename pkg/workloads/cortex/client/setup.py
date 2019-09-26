from setuptools import setup


setup(
    name="cortex",
    version="0.8.0",
    description="",
    author="Cortex Labs",
    author_email="dev@cortexlabs.com",
    install_requires=["dill==0.3.0", "requests==2.21.0"],
    setup_requires=["setuptools"],
    python_requires=[">=3.6", "==2.7.*"],
)
