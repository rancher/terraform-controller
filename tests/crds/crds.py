import kubernetes.client


class Base:
    """Base class to build CRDs off of"""

    def __init__(self, k8s_client, name, plural, namespace="default",
                 version="v1"):
        self.k8s_client = k8s_client
        self.name = name
        self.namespace = namespace
        self.plural = plural
        self.version = version
        self.group = "terraform-operator.cattle.io"
        self.crd_client = kubernetes.client.CustomObjectsApi(k8s_client)

    def create(self, body):
        self.crd_client.create_namespaced_custom_object(
            group=self.group, version=self.version, namespace=self.namespace,
            plural=self.plural, body=body)
        return

    def get(self):
        self.crd_client.get_namespaced_custom_object(
            group=self.group, version=self.version, namespace=self.namespace,
            plural=self.plural, name=self.name)
        return

    def update(self, body):
        self.crd_client.patch_namespaced_custom_object(
            group=self.group,
            version=self.version, namespace=self.namespace, plural=self.plural,
            name=self.name, body=body)
        return

    def delete(self, del_options):
        self.crd_client.delete_namespaced_custom_object(
            group=self.group, version=self.version, namespace=self.namespace,
            plural=self.plural, name=self.name, body=del_options)
        return


class Module(Base):
    """"""

    def __init__(self, url, **kwds):
        super().__init__(plural="modules", **kwds)
        self.url = url

    def create(self):
        # This needs to include the url in the body.
        return super.create()
