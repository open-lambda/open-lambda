class Handler:
    def __init__(self, name, imps, deps, mem):
        self.name = name
        self.imps = imps
        self.deps = deps
        self.mem = mem


    def get_lambda_func(self):
        imps = '\n'.join('import %s' % imp for imp in sorted(self.imps))
        simulator = '''import load_simulator
load_simulator.simulate_load(0, {mem})
'''.format(mem=self.mem)
        return '''{imps}
{simulator}
def handler(conn, event):
    try:
        return "Hello from {name}"
    except Exception as e:
        return {{'error': str(e)}}
'''.format(imps=imps, simulator=simulator, name=self.name)


    def get_packages_txt(self):
        return '\n'.join('{pkg}:{pkg}'.format(pkg=dep) for dep in sorted(self.deps))
