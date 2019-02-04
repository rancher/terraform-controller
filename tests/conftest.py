import pytest
import random
import time
import urllib3
from crds.crds import Module
from kubernetes import config

# This stops ssl warnings for unsecure certs
urllib3.disable_warnings()

DEFAULT_MODULE_URL = "https://github.com/dramich/domodule"


@pytest.fixture
def k8s_client():
    return config.new_client_from_config()


@pytest.fixture
def module_factory(k8s_client):
    """Create a module with the specified github url"""
    def _create_module(url=DEFAULT_MODULE_URL, name=random_str()):
        return Module(url=url, name=name).create()

    return _create_module


@pytest.fixture
def module(k8s_client):
    return module_factory(k8s_client)


def random_str():
    return 'random-{0}-{1}'.format(random_num(), int(time.time()))


def random_num():
    return random.randint(0, 1000000)
