.PHONY: clean

help:
	@echo "  clean           remove unwanted stuff"
	@echo "  dev             make a development package"
	@echo "  publish-test    package and upload a release to test.pypi.org"
	@echo "  publish-release package and upload a release to pypi.org"

clean:
	find . -name '*.pyc' -exec rm -f {} +
	find . -name '*.pyo' -exec rm -f {} +
	find . -name '*~' -exec rm -f {} +
	find . -name '.DS_Store' -exec rm -f {} +
	find . -name '__pycache__' -exec rm -rf {} +
	find . -name '.coverage' -exec rm -rf {} +
	rm -rf build dist *.egg-info +

dev:
	pip install .
	$(MAKE) clean

publish-test:
	python setup.py publish --test
	$(MAKE) clean

publish-release:
	python setup.py publish --release
	$(MAKE) clean
