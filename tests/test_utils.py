# -*- coding: utf-8 -*-

import string
import unittest
from shutil import rmtree
from os import remove, getenv, getcwd
from os.path import join, expanduser, isfile
from random import sample
from tempfile import gettempdir
from rtfd import __version__
from rtfd.utils import is_domain, ProjectStorage, run_cmd, check_giturl, \
    get_public_giturl, get_git_service_provider
from flask_pluginkit.utils import LocalStorage
from flask_pluginkit._compat import PY2, string_types


class UtilsTest(unittest.TestCase):

    def gen_tmpstr(self, length=8):
        return ''.join(sample(string.ascii_letters + string.digits, length))

    def test_cmd(self):
        result = run_cmd("rtfd", "-v")
        self.assertIsInstance(result, tuple)
        self.assertEqual(len(result), 3)
        _, out, _ = result
        if not PY2 and not isinstance(out, string_types):
            out = out.decode("utf-8")
        self.assertEqual(out.strip("\n"), __version__)

    def test_projectstorage(self):
        if not isfile(expanduser("~/.rtfd.cfg")):
            with self.assertRaises(AttributeError):
                ProjectStorage()

        basedir = join(gettempdir() if not getenv("TRAVIS")
                       else getcwd(), self.gen_tmpstr())
        cfg = "%s.cfg" % basedir
        exitcode, out, _ = run_cmd("rtfd", "init", "--yes", "-b", basedir,
                                   "--py3", "/usr/bin/python2", "-c", cfg)
        if exitcode != 0:
            print(out)
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

    def test_checkgiturl(self):
        self.assertFalse(check_giturl("")["status"])
        self.assertFalse(check_giturl([])["status"])
        self.assertFalse(check_giturl({})["status"])
        self.assertFalse(check_giturl(123)["status"])
        self.assertFalse(check_giturl("abc")["status"])
        self.assertFalse(check_giturl("example.com")["status"])
        self.assertFalse(check_giturl("git@github.com:staugur/rtfd")["status"])
        self.assertFalse(check_giturl("svn://gitee.com/staugur/xxx")["status"])
        self.assertFalse(check_giturl(
            "http://coding.net/staugur/xxx")["status"])
        self.assertTrue(check_giturl("http://gitee.com/staugur/xxx")["status"])
        self.assertTrue(check_giturl(
            "https://github.com/staugur/rtfd")["status"])
        private_url = "https://user:pass@github.com/staugur/rtfd"
        self.assertTrue(check_giturl(private_url)["status"])
        self.assertEqual(get_public_giturl(private_url),
                         "https://github.com/staugur/rtfd")

        self.assertEqual(
            "GitHub", get_git_service_provider("https://github.com"))
        self.assertEqual(
            "Gitee", get_git_service_provider("https://gitee.com"))
        self.assertEqual(
            "Unknown", get_git_service_provider("https://gitlab.com"))


if __name__ == '__main__':
    unittest.main()
