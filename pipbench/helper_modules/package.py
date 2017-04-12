class Package:
    def __init__(self, name, popularity, dependencies_target=None, data_file_sizes=None, install_cpu_time=None,
                 install_mem=None, import_cpu_time=None, import_mem=None):
        self.name = name
        self.dependencies_target = dependencies_target
        self.dependencies = []
        self.popularity = popularity
        self.data_file_sizes = data_file_sizes
        self.install_cpu_time = install_cpu_time
        self.install_mem = install_mem
        self.import_cpu_time = import_cpu_time
        self.import_mem = import_mem
        self.reference_count = 0

    def add_dependency(self, dependency):
        self.dependencies.append(dependency)

    def get_dependencies(self):
        return self.dependencies

    def get_dependencies_target(self):
        return self.dependencies_target

    def get_name(self):
        return self.name

    def get_popularity(self):
        return self.popularity

    def get_data_file_sizes(self):
        return self.data_file_sizes

    def get_install_cpu_time(self):
        return self.install_cpu_time

    def get_install_mem(self):
        return self.install_mem

    def get_import_cpu_time(self):
        return self.import_cpu_time

    def get_import_mem(self):
        return self.import_mem

    def should_add_more_dependencies(self):
        return len(self.dependencies) < self.dependencies_target

    def add_reference(self):
        self.reference_count += 1

    def get_reference_count(self):
        return self.reference_count
