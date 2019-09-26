from setuptools import setup, find_packages


setup(
    name="cortex",
    version="0.8.0",
    description="",
    author="Cortex Labs",
    author_email="dev@cortexlabs.com",
    install_requires=["dill==0.3.0", "requests==2.21.0"],
    setup_requires=["setuptools"],
    packages=find_packages(),
)
