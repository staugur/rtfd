#!/bin/bash
#Author:      staugur
#Version:     1.0
#Description: 最终调用的核心脚本，此脚本只负责构建，会在docs的项目下，生成不同语言和不同版本的文档
#CreateTime:  2019-08-05
#ModifyTime:  2026-09-10
#License:     BSD 3-Clause
#Copyright:   (c) 2019 by staugur.

readonly rtfd_cmd="rtfd"
rtfd_cfg="${RTFD_CFG:-$HOME/.rtfd.cfg}"
# 构建临时运行目录（用于信号清理，避免中断后残留）
RTFD_RUNTIME_TMP=""
# 配置读取缓存：bash >= 4 用关联数组；更老的 bash 退化为无缓存（直接调用），避免 declare -A 报错
RTFD_CACHE_ON=0
if [[ ${BASH_VERSINFO[0]:-0} -ge 4 ]]; then
    declare -A RTFD_CONF_CACHE=()
    RTFD_CACHE_ON=1
fi

checkExitParam() {
    local n=$1
    local c=$2
    if [[ -z "${c}" || "${c:0:1}" == "-" ]]; then
        echo "Invalid param in ${n}"
        exit 128
    fi
}

checkExitRetcode() {
    local code=$?
    if [[ $code -ne 0 ]]; then
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
    local cache_key="docs|${name}|${key}"
    local value
    if [[ $RTFD_CACHE_ON -eq 1 && -n "${RTFD_CONF_CACHE[$cache_key]+x}" ]]; then
        value="${RTFD_CONF_CACHE[$cache_key]}"
    else
        value=$($rtfd_cmd project get ${name}:${key})
        [[ $RTFD_CACHE_ON -eq 1 ]] && RTFD_CONF_CACHE[$cache_key]="$value"
    fi
    echo "${value:-$default}"
}

_getRtfdConf() {
    local config=$1
    local sfx=${config:0-4}
    local cfgfile
    if [[ "$sfx" == ".cfg" || "$sfx" == ".ini" ]] && [[ -f $config ]]; then
        cfgfile=$config
        shift
    else
        cfgfile=$rtfd_cfg
    fi
    local name=$1
    local key=$2
    local default=$3
    local cache_key="cfg|${cfgfile}|${name}|${key}"
    local value
    if [[ $RTFD_CACHE_ON -eq 1 && -n "${RTFD_CONF_CACHE[$cache_key]+x}" ]]; then
        value="${RTFD_CONF_CACHE[$cache_key]}"
    else
        value=$($rtfd_cmd cfg -c "$cfgfile" ${name} ${key})
        [[ $RTFD_CACHE_ON -eq 1 ]] && RTFD_CONF_CACHE[$cache_key]="$value"
    fi
    echo "${value:-$default}"
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
    local project_python=$5

    local rtfd_server=$(_getRtfdConf api server_url)
    local default_index=$(_getRtfdConf py index https://pypi.org/simple)

    #: 校验参数（server_url 必须已配置，由 rtfd 服务端统一负责归一化/告警）
    checkExitParam _envManager_project_runtime_dir $project_runtime_dir
    checkExitParam _envManager_project_docs_dir $project_docs_dir
    checkExitParam _envManager_branch $branch
    checkExitParam _envManager_project_name $project_name
    checkExitParam _envManager_rtfd_server $rtfd_server
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
    local py_version=${py_version:=$(_getDocsConf $project_name Version 3)}
    local py_requirements=${py_requirements:=$(_getDocsConf $project_name Requirement)}
    local py_install_project=${py_install_project:=$(_getDocsConf $project_name Install false)}
    local py_index=${py_index:=$(_getDocsConf $project_name Index $default_index)}
    #: 防路径穿越：sourcedir 必须是相对路径，且任何路径段都不得是 ..（防止跳出项目根目录）
    if [[ "${sphinx_sourcedir:0:1}" == "/" ]]; then
        echo "In rtfd.ini, sourcedir cannot be an absolute path"
        exit 1
    fi
    if [[ "$sphinx_sourcedir" == ".." || "$sphinx_sourcedir" == ".."/* || "$sphinx_sourcedir" == *"/.." || "$sphinx_sourcedir" == *"/../"* ]]; then
        echo "In rtfd.ini, sourcedir cannot contain '..' path segment"
        exit 1
    fi
    #: python解释器：优先使用rtfd传入的，否则按版本号从配置中取
    local py_path=${project_python:-$(_getRtfdConf py ${py_version})}
    if [[ -z "$py_path" ]]; then
        py_version=$(_getRtfdConf py default)
        py_path=$(_getRtfdConf py ${py_version})
    fi
    checkExitParam _envManager_py_path $py_path
    command -v "$py_path" &>/dev/null
    checkExitRetcode
    #: 检查Sphinx配置文件
    local sphinx_conf=$(_joinPath $sphinx_sourcedir conf.py)
    if [ ! -f $sphinx_conf ]; then
        echo "Not found docs conf.py in $(_joinPath $project_runtime_dir $sphinx_sourcedir)"
        exit 1
    fi
    #: 创建虚拟环境
    local vd="venv-${py_version}"
    local venv=( "$py_path" -m virtualenv )
    if [ ! -d "$vd" ]; then
        "${venv[@]}" "$vd"
        checkExitRetcode
    fi
    #: 激活虚拟环境
    source ${vd}/bin/activate
    checkExitRetcode
    #: 安装依赖（必须自行将sphinx写入依赖包文件）
    local venv_py
    venv_py=$(_joinPath "$project_runtime_dir" "${vd}/bin/python")
    local venv_pip_install=( "$venv_py" -m pip install -i "$py_index" )
    for req in ${py_requirements//,/ }; do
        "${venv_pip_install[@]}" -r "$req"
        checkExitRetcode
    done
    if [[ "${py_install_project}" == "true" || "${py_install_project}" == "True" ]]; then
        "${venv_pip_install[@]}" .
        checkExitRetcode
    fi
    #: 更新conf.py
    cat >>$sphinx_conf <<EOF
#: Automatic generated by rtfd at $(date '+%Y-%m-%d %H:%M:%S')
# 以 data-* 属性注入 rtfd.js 挂件配置（name / branch / api），
# 脚本通过 document.currentScript 读取这些属性，免去 query 串拼接，更直观。
# 要求 Sphinx >= 1.8（add_js_file 支持 **attributes）。
_rtfd_js_url = '${rtfd_server}/rtfd/assets/rtfd.js?v=$($rtfd_cmd -v)'
def _rtfd_add_widget(app):
    _rtfd_attrs = {
        'data-name': '${project_name}',
        'data-branch': '${branch}',
        'data-api': '${rtfd_server}',
    }
    if hasattr(app, 'add_js_file'):
        app.add_js_file(_rtfd_js_url, **_rtfd_attrs)
    elif hasattr(app, 'add_javascript'):
        app.add_javascript(_rtfd_js_url)
# 与项目自身可能存在的 setup() 兼容组合，避免覆盖
_rtfd_orig_setup = globals().get('setup')
def setup(app):
    if callable(_rtfd_orig_setup):
        _rtfd_orig_setup(app)
    _rtfd_add_widget(app)
EOF
    #: 构建
    local sphinx_build=$(_joinPath $project_runtime_dir ${vd}/bin/sphinx-build)
    for lang in ${sphinx_languages//,/ }; do
        local project_docs_lang_dir=$(_joinPath ${project_docs_dir} ${lang})
        $sphinx_build -E -T -D language=${lang} -b ${sphinx_builder} $sphinx_sourcedir $(_joinPath ${project_docs_lang_dir} ${branch})
        checkExitRetcode
        ln -nsf $(_joinPath ${project_docs_lang_dir} ${project_latest}) $(_joinPath ${project_docs_lang_dir} latest)
        checkExitRetcode
    done
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
    -p, --python  The python command for building, default is resolved by rtfd config
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
        -p | --python)
            local python="${2}"
            checkExitParam python $python
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
    command -v "$rtfd_cmd" >/dev/null 2>&1
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
    local runtimes_tmp=$(mktemp -d -p $runtimes_dir)
    RTFD_RUNTIME_TMP=$runtimes_tmp

    _codeManager $project_name $branch $runtimes_tmp
    checkExitRetcode

    local project_docs_dir=$(_joinPath $docs_dir $project_name)
    local project_runtime_dir=$(_joinPath $runtimes_tmp $project_name)
    _envManager $project_name $branch $project_runtime_dir $project_docs_dir $python
    checkExitRetcode

    local utime=$(($SECONDS - $stime))
    echo "Build Successfully, $utime seconds passed."
    rm -rf "$runtimes_tmp"
    exit 0
}

Clean() {
    echo "The program was terminated, will exit!"
    [[ -n "$RTFD_RUNTIME_TMP" ]] && rm -rf "$RTFD_RUNTIME_TMP"
    exit 1
}

trap 'Clean; exit' SIGINT SIGTERM

main "$@"
