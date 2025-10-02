#!/bin/bash

# 检查参数数量
if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <port> <target_ip1> <target_ip2> ... <target_ipN>"
  exit 1
fi

# 获取端口和目标IP列表
PORT="$1"
shift
TARGETS=("$@")

# 定义执行tcping的函数
run_tcping() {
  local TARGET="$1"
  local PORT="$2"
  local TEMP_FILE=$(mktemp)

  # 执行 60 次 tcping
  for i in {1..60}; do
      tcping -x 1 $TARGET $PORT >> $TEMP_FILE
      sleep 1
  done

  # 提取时间值并计算统计数据
  awk -v target="$TARGET" '
  BEGIN {
    min = -1
    max = 0
    sum = 0
    count = 0
    rcv = 0
  }
  /<syn,ack>/ {
    match($0, /([0-9]+\.[0-9]+) ms/, arr)
    if (arr[1] != "") {
      time = arr[1]
      times[count++] = time
      sum += time
      if (min == -1 || time < min) min = time
      if (time > max) max = time
      rcv++
    }
  }
  END {
    if (count > 0) {
      avg = sum / count
      loss = (60 - rcv) / 60 * 100
      printf "%s : xmt/rcv/%%loss = 60/%d/%.0f%%, min/avg/max = %.2f/%.2f/%.2f\n", target, rcv, loss, min, avg, max
    } else {
      print "没有有效的响应时间"
    }
  }' $TEMP_FILE

  # 删除临时文件
  rm -f $TEMP_FILE
}

# 遍历所有目标IP并启动后台任务
for TARGET in "${TARGETS[@]}"; do
  run_tcping "$TARGET" "$PORT" &
done

# 等待所有后台任务完成
wait

echo "所有检测任务已完成。"

