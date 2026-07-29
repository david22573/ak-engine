#!/usr/bin/env bash
# historian_env.sh — Centralized AK_HISTORIAN_WORKDIR environment resolution for ak-engine scripts

if [ -z "${AK_HISTORIAN_WORKDIR:-}" ]; then
    if [ -d "${HOME}/Github/ak-historian/.ak-historian/work" ]; then
        export AK_HISTORIAN_WORKDIR="${HOME}/Github/ak-historian/.ak-historian/work"
    elif [ -d "../ak-historian/.ak-historian/work" ]; then
        export AK_HISTORIAN_WORKDIR="../ak-historian/.ak-historian/work"
    else
        export AK_HISTORIAN_WORKDIR=".ak-historian/work"
    fi
fi
export HISTORIAN_WORKDIR="${AK_HISTORIAN_WORKDIR}"
