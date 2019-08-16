# rtfd
Build, read your exclusive and fuck docs.

## 设计思路

根据readthedocs.io的访问效果，例如
    https://flask-pluginkit.readthedocs.io/en/v3.3.0/
    https://flask-pluginkit.readthedocs.io/zh_CN/latest/

所以可以简单概括为：
    {{ Domain }}/{{ Lang }}/{{ branch }}

对应的html目录层次为：
    {{ Docs_Build_HTML }}/{{ Lang }}/{{ branch}}

可以有多个文档项目，支持翻译项目。

github webhook触发构建；安装依赖（临时venv）；branch中latest链接到master

一个命令，通过pip安装，在用户下创建/读取配置文件定义一个根目录，命令所运行期间的操作在根目录中。

## 实现

.. 前提，脚本一键生成依赖环境和构建入口，校验SHELL类型及版本

.. 构建核心

1. 处理动作

- 新增文档
    core/create.py 参数
    - 仓库名和地址，在runtimes检出，并在docs创建唯一项目
    - 读取文档额外配置，也支持读取yml的配置文件（优先级低）
    - 在docs项目中保存配置，用flask-pluginkit的localstorage

- 更新文档

- 删除文档

2. 触发构建
    core/build.py 参数
    - python, venv, 不同版本
    - sphinx-build
    - 仓库
    - 分支
    - 语言

.. api接收webhook触发构建


## 示例
```

exec="/usr/bin/python2.7"
cmd="/usr/local/bin/sphinx-build"
src="docs"
dst="runtime/html/"
branch="latest"

1. en

lang=en
$exec $cmd -E -T -D language=$lang -b html $src ${dst}/${lang}/${branch}

for tag v3.3.0
branch=v3.3.0
git checkout $branch
$exec

2. cn
lang=zh_CN

$exec $cmd -E -T -D language=$lang -b html $src ${dst}/${lang}/${branch}

```

## 参考
- https://amito.me/2018/Using-SH-in-Python/
- https://debugtalk.com/post/use-pyenv-manage-multiple-python-virtualenvs/
- https://blog.csdn.net/guoqianqian5812/article/details/68610760
