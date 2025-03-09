#!/bin/bash

# 检查是否提供了提交消息
if [ -z "\$1" ]; then
  echo "错误：请提供提交消息作为参数。"
  echo "用法: ./push.sh \"提交消息\""
  exit 1
fi

# 获取提交消息
commit_message="\$1"

# 执行 Git 操作
git add .
git commit -m "$commit_message"
git push origin master
