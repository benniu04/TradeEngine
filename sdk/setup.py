from setuptools import setup, find_packages

setup(
    name="tradeengine",
    version="0.1.0",
    packages=find_packages(),
    install_requires=["requests>=2.28"],
    python_requires=">=3.8",
    description="Python SDK for TradeEngine",
)
