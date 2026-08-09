from setuptools import setup, find_packages

setup(
    name="renos-sdk",
    version="1.0.0",
    description="RenOS API Python SDK",
    packages=find_packages(),
    python_requires=">=3.10",
    install_requires=[
        "requests>=2.31.0",
        "websockets>=12.0",
        "pydantic>=2.5.0",
    ],
)
