# -*- coding: utf-8 -*-

import io
import os
import re
import ast
import unittest
from setuptools import setup, Command


def test_suite():
    test_loader = unittest.TestLoader()
    test_suite = test_loader.discover('tests', pattern='test_*.py')
    return test_suite


def _get_version():
    version_re = re.compile(r'__version__\s+=\s+(.*)')

    with open('rtfd/__init__.py', 'rb') as fh:
        version = ast.literal_eval(version_re.search(
            fh.read().decode('utf-8')).group(1))

    return str(version)


def _get_author():
    author_re = re.compile(r'__author__\s+=\s+(.*)')
    mail_re = re.compile(r'(.*)\s<(.*)>')

    with open('rtfd/__init__.py', 'rb') as fh:
        author = ast.literal_eval(author_re.search(
            fh.read().decode('utf-8')).group(1))

    return (mail_re.search(author).group(1), mail_re.search(author).group(2))


def _get_readme():
    with io.open('README.rst', 'rt', encoding='utf8') as f:
        return f.read()


class PublishCommand(Command):

    description = "Publish a new version to pypi"

    user_options = [
        # The format is (long option, short option, description).
        ("test", None, "Publish to test.pypi.org"),
        ("release", None, "Publish to pypi.org"),
    ]

    def initialize_options(self):
        """Set default values for options."""
        self.test = False
        self.release = False

    def finalize_options(self):
        """Post-process options."""
        if self.test:
            print("V%s will publish to the test.pypi.org" % version)
        elif self.release:
            print("V%s will publish to the pypi.org" % version)

    def run(self):
        """Run command."""
        os.system("pip install -U setuptools twine wheel")
        os.system("rm -rf build/ dist/ rtfd.egg-info/")
        os.system("python setup.py sdist bdist_wheel")
        if self.test:
            os.system(
                "twine upload --repository-url https://test.pypi.org/legacy/ dist/*")
        elif self.release:
            os.system("twine upload dist/*")
        os.system("rm -rf build/ dist/ rtfd.egg-info/")
        if self.test:
            print("V%s publish to the test.pypi.org successfully" % version)
        elif self.release:
            print("V%s publish to the pypi.org successfully" % version)
        exit()


version = _get_version()
(author, email) = _get_author()
setup(
    name='rtfd',
    version=version,
    url='https://github.com/staugur/rtfd',
    download_url="https://github.com/staugur/rtfd/releases/tag/%s" % version,
    project_urls={
        "Documentation": "https://docs.saintic.com/rtfd/",
        "Code": "https://github.com/staugur/rtfd",
        "Issue tracker": "https://github.com/staugur/rtfd/issues",
    },
    license='BSD 3-Clause',
    author=author,
    author_email=email,
    keywords=["docs", "sphinx", "reStructuredText"],
    description='Build, read your exclusive and fuck docs.',
    long_description=_get_readme(),
    test_suite='setup.test_suite',
    packages=['rtfd', ],
    include_package_data=True,
    zip_safe=False,
    entry_points={
        'console_scripts': [
            'rtfd = rtfd._cli:cli'
        ]
    },
    install_requires=[
        'Flask-PluginKit>=3.5.0',
        'Click>=7.0',
        'configparser'
    ],
    python_requires=">=2.7, !=3.0.*, !=3.1.*, !=3.2.*, !=3.3.*, !=3.4.*",
    cmdclass={
        'publish': PublishCommand,
    },
    classifiers=[
        'Development Status :: 4 - Beta',
        'Environment :: Console',
        'Intended Audience :: Developers',
        'License :: OSI Approved :: BSD License',
        'Programming Language :: Python :: 2.7',
        'Programming Language :: Python :: 3.5',
        'Programming Language :: Python :: 3.6',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: Implementation :: CPython',
        'Programming Language :: Python :: Implementation :: PyPy',
        'Topic :: Software Development :: Libraries :: Python Modules'
    ]
)
