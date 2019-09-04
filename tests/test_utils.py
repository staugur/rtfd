# -*- coding: utf-8 -*-

import string
import unittest
from shutil import rmtree
from os import remove
from os.path import join
from random import sample
from tempfile import gettempdir
from rtfd import __version__
from rtfd.utils import is_domain, ProjectStorage, run_cmd
from flask_pluginkit.utils import LocalStorage


class UtilsTest(unittest.TestCase):

    def gen_tmpstr(self, length=8):
        return ''.join(sample(string.ascii_letters + string.digits, length))

    def test_cmd(self):
        result = run_cmd("rtfd", "-v")
        self.assertIsInstance(result, tuple)
        self.assertEqual(len(result), 3)
        _, out, _ = result
        self.assertEqual(out.strip("\n"), __version__)

    def test_projectstorage(self):
        basedir = join(gettempdir(), self.gen_tmpstr())
        cfg = "%s.cfg" % basedir
        run_cmd("rtfd", "init", "--yes", "-b", basedir,
                "--py3", "/usr/bin/python2", "-c", cfg)
        storage = ProjectStorage(cfg)
        self.assertIsInstance(storage, LocalStorage)
        #: Refer to the test of flask-pluginkit
        data = dict(a=1, b=2)
        storage.set('test', data)
        newData = storage.get('test')
        self.assertIsInstance(newData, dict)
        self.assertEqual(newData, data)
        self.assertEqual(len(storage), len(storage.list))
        # test setitem getitem
        storage["test"] = "hello"
        self.assertEqual("hello", storage["test"])
        # Invalid, LocalStorage did not implement this method
        del storage["test"]
        self.assertEqual("hello", storage["test"])
        self.assertIsNone(storage['_non_existent_key_'])
        self.assertEqual(1, storage.get('_non_existent_key_', 1))
        # test other index
        storage.index = '_non_existent_index_'
        self.assertEqual(0, len(storage))
        #: after
        rmtree(basedir)
        remove(cfg)

    def test_checkdomain(self):
        self.assertFalse(is_domain('http://127.0.0.1'))
        self.assertFalse(is_domain('http://localhost:5000'))
        self.assertFalse(is_domain('https://abc.com'))
        self.assertFalse(is_domain('https://abc.com:8443'))
        self.assertFalse(is_domain('ftp://192.168.1.2'))
        self.assertFalse(is_domain('rsync://192.168.1.2'))
        self.assertFalse(is_domain('192.168.1.2'))
        self.assertFalse(is_domain('1.1.1.1'))
        self.assertFalse(is_domain('localhost'))
        self.assertFalse(is_domain('127.0.0.1:8000'))
        self.assertFalse(is_domain('://127.0.0.1/hello-world'))
        self.assertFalse(is_domain("x_y_z.com"))
        self.assertFalse(is_domain("_x-y-z.com"))
        self.assertFalse(is_domain("false"))
        self.assertTrue(is_domain('test.test.example.com'))
        self.assertTrue(is_domain("x-y-z.com"))
        self.assertTrue(is_domain("abc.com"))
        self.assertTrue(is_domain("localhost.localdomain"))


if __name__ == '__main__':
    unittest.main()
