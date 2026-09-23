# Runtime fixtures

These frozen YAML graphs preserve the independently reviewed inputs of existing CLI, state-replay, output-invalidation and route-matrix tests at 1b1fdcde. Skill files are inert stubs; production instructions, scripts, documents and images are excluded. The fixture IDs intentionally remain stable for existing State/Event and Golden cases. Configuration publications never update this directory. They are not runtime defaults or a network fallback. Add or reduce only the graph features required by a concrete runtime test.

Run production configuration checks against a separately downloaded, explicitly versioned set with scripts/validate-configuration.py.
