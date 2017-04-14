class Handler:
    def __init__(self, name, dependencies_target):
        self.name = name
        self.dependencies_target = dependencies_target
        self.dependencies = []

    def add_dependency(self, dependency):
        self.dependencies.append(dependency)

    def get_name(self):
        return self.name

    def get_dependencies(self):
        return self.dependencies

    def should_add_more_dependencies(self):
        return len(self.dependencies) < self.dependencies_target