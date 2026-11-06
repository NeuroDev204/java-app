#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "=== Đang xóa thông tin tài khoản và mật khẩu đã lưu ==="

python3 -c '
import json

targets = [
    "rms_data.json",
    "pkg/jvm/rms_data.json",
    "pkg/vm/default_rms.json",
    "mobile/assets/rms_data.json"
]

to_remove = {
    "user_pass",
    "x2_user_pass",
    "isIndexServer",
    "x2_isIndexServer",
    "selectedServerHost",
    "listServer",
}

for path in targets:
    try:
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)
        before_len = len(data)
        new_data = [s for s in data if s["name"] not in to_remove]
        with open(path, "w", encoding="utf-8") as f:
            json.dump(new_data, f, indent=2)
        print(f"-> {path}: đã xóa {before_len - len(new_data)} store (còn {len(new_data)} store)")
    except Exception as e:
        print(f"-> {path}: bỏ qua ({e})")
'

echo "=== Hoàn tất! Lần mở game tiếp theo sẽ ở trạng thái tài khoản mới ==="
