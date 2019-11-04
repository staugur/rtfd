rtfd
====

Build, read your exclusive and fuck docs.

|Build Status| |Documentation Status| |codecov| |PyPI|

Installation
------------

- Production Version

    .. code-block:: bash

        $ pip install -U rtfd

- Development Version

    .. code-block:: bash

        $ pip install -U git+https://github.com/staugur/rtfd.git@master

Quickstart
----------

1. rtfd init -b your_data_dir --other-options

2. rtfd project -a create --url git-url --other-options your-docs-project

3. rtfd build your-docs-project

More options with ``--help / -h`` option.

Documentation
-------------

More please see the `detailed documentation <https://docs.saintic.com/rtfd>`_

.. |Documentation Status| image:: https://open.saintic.com/rtfd/badge/saintic-docs
    :target: https://docs.saintic.com/rtfd/

.. |Build Status| image:: https://travis-ci.org/staugur/rtfd.svg?branch=master
    :target: https://travis-ci.org/staugur/rtfd

.. |codecov| image:: https://codecov.io/gh/staugur/rtfd/branch/master/graph/badge.svg
    :target: https://codecov.io/gh/staugur/rtfd

.. |PyPI| image:: https://img.shields.io/pypi/v/rtfd.svg?style=popout
    :target: https://pypi.org/project/rtfd/
