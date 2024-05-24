#!/bin/bash
#Author:      staugur
#Version:     0.5
#Description: 最终调用的核心脚本，此脚本只负责构建，会在docs的项目下，生成不同语言和不同版本的文档
#CreateTime:  2019-08-05
#ModifyTime:  2022-01-06
#License:     BSD 3-Clause
#Copyright:   (c) 2019 by staugur.

readonly rtfd_cmd="rtfd"
rtfd_cfg="${RTFD_CFG:-$HOME/.rtfd.cfg}"

checkExitParam() {
    local n=$1
    local c=$2
    if [[ -z "${c}" || "x" == "x${c}" || "${c:0:1}" == "-" ]]; then
        echo "Invalid param in ${n}"
        exit 128
    fi
}

checkExitRetcode() {
    local code=$?
    if [[ $code -ne 0 || "${code}" != "0" ]]; then
        echo "Command sending error code $code in $(pwd), the traceback stack:"
        echo "    $(caller 0)"
        echo "    $(caller 1)"
        exit 128
    fi
}

_getDocsConf() {
    local name=$1
    local key=$2
    local default=$3
    local value=$($rtfd_cmd project get ${name}:${key})
    echo ${value:-$default}
}

_getRtfdConf() {
    local config=$1
    local sfx=${config:0-4}
    if [[ "$sfx" == ".cfg" || "$sfx" == ".ini" ]]; then
        local is_cfg="yes"
    fi
    if [[ -f $config && "$is_cfg" == "yes" ]]; then
        shift
        cmd="$rtfd_cmd cfg -c $config"
    else
        cmd="$rtfd_cmd cfg -c $rtfd_cfg"
    fi
    local name=$1
    local key=$2
    local default=$3
    local value=$($cmd ${name} ${key})
    echo ${value:-$default}
}

_joinPath() {
    echo "${1:+$1/}$2" | sed 's#//#/#g'
}

_debugp() {
    local log_level=$(_getRtfdConf default log_level)
    if [[ "$log_level" == "debug" || "$log_level" == "DEBUG" ]]; then
        echo -e "$@"
    fi
}

_envManager() {
    #: 切换到项目中，创建虚拟环境并构建文档
    local project_name=$1
    local branch=$2
    local project_runtime_dir=$3
    local project_docs_dir=$4

    local py2_path=$(_getRtfdConf py py2)
    local py3_path=$(_getRtfdConf py py3)
    local rtfd_server=$(_getRtfdConf api server_url)
    local server_static_url=$(_getRtfdConf api server_static_url ${rtfd_server}/rtfd/assets/)
    local favicon_url=$(_getRtfdConf default favicon_url https://static.saintic.com/rtfd/favicon.png)
    local default_index=$(_getRtfdConf py index https://pypi.org/simple)

    #: 校验参数
    checkExitParam _envManager_py2_path $py2_path
    checkExitParam _envManager_py3_path $py3_path
    which $py2_path &>/dev/null
    checkExitRetcode
    which $py3_path &>/dev/null
    checkExitRetcode
    checkExitParam _envManager_project_runtime_dir $project_runtime_dir
    checkExitParam _envManager_project_docs_dir $project_docs_dir
    checkExitParam _envManager_branch $branch
    checkExitParam _envManager_project_name $project_name
    checkExitParam _envManager_rtfd_server $rtfd_server
    checkExitParam _envManager_server_static_url $server_static_url
    checkExitParam _envManager_favicon_url $favicon_url
    checkExitParam _envManager_default_index $default_index

    cd ${project_runtime_dir}
    checkExitRetcode

    #: 尝试读取项目仓库根目录下的文档配置文件
    project_ini=".rtfd.ini"
    if [ -f $project_ini ]; then
        local project_latest=$(_getRtfdConf $project_ini project latest)
        local sphinx_sourcedir=$(_getRtfdConf $project_ini sphinx sourcedir)
        local sphinx_languages=$(_getRtfdConf $project_ini sphinx lang)
        local sphinx_builder=$(_getRtfdConf $project_ini sphinx builder)
        local py_version=$(_getRtfdConf $project_ini python version)
        local py_requirements=$(_getRtfdConf $project_ini python requirement)
        local py_install_project=$(_getRtfdConf $project_ini python install)
        local py_index=$(_getRtfdConf $project_ini python index)
    fi
    local project_latest=${project_latest:=$(_getDocsConf $project_name Latest master)}
    local sphinx_sourcedir=${sphinx_sourcedir:=$(_getDocsConf $project_name SourceDir docs)}
    local sphinx_languages=${sphinx_languages:=$(_getDocsConf $project_name Lang en)}
    local sphinx_builder=${sphinx_builder:=$(_getDocsConf $project_name Builder html)}
    local py_version=${py_version:=$(_getDocsConf $project_name Version 2)}
    local py_requirements=${py_requirements:=$(_getDocsConf $project_name Requirement)}
    local py_install_project=${py_install_project:=$(_getDocsConf $project_name Install false)}
    local py_index=${py_index:=$(_getDocsConf $project_name Index $default_index)}
    if [[ "${sphinx_sourcedir:0:1}" == "/" || "${sphinx_sourcedir:0:2}" == ".." ]]; then
        echo "In rtfd.ini, sourcedir cannot start with / or .."
        exit 1
    fi
    case $py_version in
    2)
        local py_path=$py2_path
        ;;
    3)
        local py_path=$py3_path
        ;;
    *)
        local py_path=$py3_path
        ;;
    esac
    local vd="venv-${py_version}"
    local venv="${py_path} -m virtualenv"
    #: 创建虚拟环境
    if [ ! -d $vd ]; then
        $venv $vd
        checkExitRetcode
    fi
    #: 激活虚拟环境
    source ${vd}/bin/activate
    checkExitRetcode
    #: 安装依赖（必须自行将sphinx写入依赖包文件）
    local venv_py=$(_joinPath $project_runtime_dir ${vd}/bin/python)
    local venv_pip_install="${venv_py} -m pip install -i ${py_index}"
    for req in ${py_requirements//,/ }; do
        $venv_pip_install -r $req
        checkExitRetcode
    done
    if [[ "${py_install_project}" == "true" || "${py_install_project}" == "True" ]]; then
        $venv_pip_install .
        checkExitRetcode
    fi
    #: 更新conf.py
    local sphinx_conf=$(_joinPath $sphinx_sourcedir conf.py)
    if [ ! -f $sphinx_conf ]; then
        echo "Not found docs conf.py in $(_joinPath $project_runtime_dir $sphinx_sourcedir)"
        exit 1
    fi
    cat >>$sphinx_conf <<EOF
#: Automatic generated by rtfd at $(date '+%Y-%m-%d %H:%M:%S')
if not 'html_js_files' in globals():
    html_js_files = []
html_js_files.append("${server_static_url}rtfd.js?v=$($rtfd_cmd -v)&name=${project_name}&branch=${branch}&rtfd_api=${rtfd_server}")
if 'html_favicon' not in globals():
    html_favicon = '${favicon_url}'
EOF
    #: 执行构建前的钩子命令：
    local before_hook=$(_getDocsConf $project_name BeforeHook)
    if [ ! -z "$before_hook" ]; then
        _debugp "Trigger before_hook: ${before_hook}"
        ($before_hook)
        checkExitRetcode
    fi
    #: 构建
    local sphinx_build=$(_joinPath $project_runtime_dir ${vd}/bin/sphinx-build)
    for lang in ${sphinx_languages//,/ }; do
        local project_docs_lang_dir=$(_joinPath ${project_docs_dir} ${lang})
        $sphinx_build -E -T -D language=${lang} -b ${sphinx_builder} $sphinx_sourcedir $(_joinPath ${project_docs_lang_dir} ${branch})
        checkExitRetcode
        ln -nsf $(_joinPath ${project_docs_lang_dir} ${project_latest}) $(_joinPath ${project_docs_lang_dir} latest)
        checkExitRetcode
    done
    #: 执行构建成功后的钩子命令：
    local after_hook=$(_getDocsConf $project_name AfterHook)
    if [ ! -z "$after_hook" ]; then
        _debugp "Trigger after_hook: ${after_hook}"
        ($after_hook) && _debugp "after_hook ok" || _debugp "after_hook fail"
    fi
    #: 退出虚拟环境
    deactivate
    #: 后续处理：依照${project_ini}更新项目信息
    if [ -f $project_ini ]; then
        $rtfd_cmd project update -f $project_ini $project_name
    fi
    return 0
}

_codeManager() {
    #: 克隆指定分支代码并切换项目中
    local project_name=$1
    local branch=$2
    local runtime_dir=$3
    local project_git=$(_getDocsConf $project_name URL)
    checkExitParam _codeManager_project_name $project_name
    checkExitParam _codeManager_project_git $project_git
    checkExitParam _codeManager_branch $branch
    checkExitParam _codeManager_runtime_dir $runtime_dir
    cd $runtime_dir
    checkExitRetcode
    [ -d $project_name ] && rm -rf $project_name
    git clone --branch $branch --single-branch --depth=1 --recursive $project_git $project_name
    checkExitRetcode
    cd $project_name
    checkExitRetcode
}

usage() {
    printf "
Usage: $0 [options]

Options:

    -h, --help    The help information
    -n, --name    The docs project name
    -b, --branch  The docs project branch, default is master.
    -c, --config  The config file, default is ${rtfd_cfg}
"
    return $?
}

main() {
    local stime=$SECONDS
    if [ $# -eq 0 ]; then
        usage
        exit 1
    fi
    while [ $# -gt 0 ]; do
        case "$1" in
        -n | --name)
            local project_name="${2,,}"
            checkExitParam project_name $project_name
            shift
            ;;
        -b | --branch)
            local branch="${2}"
            checkExitParam branch $branch
            shift
            ;;
        -c | --config)
            local config="${2}"
            checkExitParam config $config
            rtfd_cfg="${config:=$rtfd_cfg}"
            shift
            ;;
        -h | --help | \?)
            usage
            exit 0
            ;;
        --)
            shift
            break
            ;;
        *)
            break
            ;;
        esac
        shift
    done
    if [ ! -f $rtfd_cfg ]; then
        echo "Not found config file $rtfd_cfg"
        exit 1
    fi
    which $rtfd_cmd > /dev/null 2>&1
    [ $? -ne 0 ] && echo "Not found $rtfd_cmd" && exit 130
    #: 设置默认配置
    local branch=${branch:=master}
    local base_dir=$(_getRtfdConf default base_dir)
    echo "Run a build for ${project_name}:${branch} with rtfd $($rtfd_cmd -v) at $(date +%FT%T)"
    #: 校验参数
    checkExitParam base_dir $base_dir
    checkExitParam project_name $project_name
    checkExitParam branch $branch
    if [[ ${#base_dir} -lt 2 || "${base_dir:0:1}" != "/" ]]; then
        echo "invalid base_dir"
        exit 1
    fi
    test -d $base_dir
    checkExitRetcode

    local docs_dir=$(_joinPath $base_dir docs)
    local runtimes_dir=$(_joinPath $base_dir runtimes)
    [ -d $docs_dir ] || mkdir -p $docs_dir
    [ -d $runtimes_dir ] || mkdir -p $runtimes_dir
    local runtimes_dir=$(mktemp -d -p $runtimes_dir)

    _codeManager $project_name $branch $runtimes_dir
    checkExitRetcode

    local project_docs_dir=$(_joinPath $docs_dir $project_name)
    local project_runtime_dir=$(_joinPath $runtimes_dir $project_name)
    _envManager $project_name $branch $project_runtime_dir $project_docs_dir
    checkExitRetcode

    local utime=$(($SECONDS - $stime))
    echo "Build Successfully, $utime seconds passed."
    rm -rf $runtimes_dir
    exit 0
}

Clean() {
    echo "The program was terminated, will exit!"
    exit 1
}

trap 'Clean; exit' SIGINT SIGTERM

main "$@"
