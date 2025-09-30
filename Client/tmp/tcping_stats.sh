#!/bin/bash

# 检查参数数量
if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <target_ip> <port>"
  exit 1
fi

# 目标地址和端口
TARGET="$1"
PORT="$2"

# 临时文件存储 tcping 输出
TEMP_FILE=$(mktemp)

# 执行 60 次 tcping
for i in {1..60}; do
  tcping -x 1 $TARGET $PORT >> $TEMP_FILE
  sleep 1
done

# 提取时间值并计算统计数据
awk -v target="$TARGET" '
BEGIN {
  min = 0
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
    if (time < min || min == 0) min = time
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
