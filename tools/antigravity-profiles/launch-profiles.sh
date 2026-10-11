#!/usr/bin/env bash
# ==============================================================================
# launch-profiles.sh - Antigravity Multi-Profile Launcher (Bash / macOS / Linux / WSL)
# ==============================================================================
# รองรับการรัน Antigravity CLI (agy) แยก Profile อย่างเด็ดขาด
# ทั้ง gcloud, .gemini config, credentials, cache และ workspace
# ==============================================================================

set -euo pipefail

# ไดเรกทอรีหลักสำหรับจัดเก็บ config แยกแต่ละ Profile
BASE_CONFIG_DIR="${BASE_CONFIG_DIR:-$HOME/.antigravity-profiles}"

# ตรวจสอบคำสั่ง agy หรือ fallback ไปที่ antigravity
AGY_BIN="$(command -v agy || command -v antigravity || true)"
if [ -z "$AGY_BIN" ]; then
  # ตรวจสอบตำแหน่ง default เพิ่มเติม
  if [ -f "$HOME/.local/bin/agy" ]; then
    AGY_BIN="$HOME/.local/bin/agy"
  elif [ -f "$HOME/AppData/Local/agy/bin/agy.exe" ]; then
    AGY_BIN="$HOME/AppData/Local/agy/bin/agy.exe"
  else
    echo "Error: ไม่พบคำสั่ง 'agy' ใน PATH"
    echo "กรุณาตรวจสอบการติดตั้ง Antigravity CLI"
    exit 1
  fi
fi

# ฟังก์ชันเตรียม Environment สำหรับ Profile
prepare_profile_env() {
  local profile_name="$1"
  local profile_dir="$BASE_CONFIG_DIR/$profile_name"

  mkdir -p "$profile_dir/gcloud"
  mkdir -p "$profile_dir/home/.gemini"

  # คืนค่า path สำหรับ export
  echo "$profile_dir"
}

# ฟังก์ชันสร้างคำสั่งสำหรับ instance
create_instance_cmd() {
  local profile_name="$1"
  local workspace="$2"
  local prompt_text="${3:-}" # หากระบุจะทำงานในโหมด Non-interactive (Headless สำหรับ Agent)

  local profile_dir="$BASE_CONFIG_DIR/$profile_name"

  cat <<EOF
export CLOUDSDK_CONFIG="$profile_dir/gcloud"
export HOME="$profile_dir/home"
export USERPROFILE="$profile_dir/home"
export ANTIGRAVITY_EXECUTABLE_DATA_DIR="$profile_dir/home/.gemini/antigravity"

echo "=================================================="
echo " Starting Antigravity Instance: $profile_name"
echo " Isolated Config Home: \$HOME"
echo " gcloud Config:        \$CLOUDSDK_CONFIG"
echo " Workspace:            $workspace"
echo "=================================================="

mkdir -p "$workspace"
cd "$workspace"

if [ -n "$prompt_text" ]; then
  # โหมด Headless สำหรับ Agent AI เรียกใช้งาน
  "$AGY_BIN" -p "$prompt_text" --output-format json --dangerously-skip-permissions
else
  # โหมด Interactive สำหรับทำงานใน Terminal
  "$AGY_BIN"
fi
EOF
}

# ตัวอย่างการตั้งค่า Workspaces
WORKSPACE_1="${WORKSPACE_1:-$HOME/projects/workspace-1}"
WORKSPACE_2="${WORKSPACE_2:-$HOME/projects/workspace-2}"

prepare_profile_env "acc1" > /dev/null
prepare_profile_env "acc2" > /dev/null

CMD_ACC1="$(create_instance_cmd "acc1" "$WORKSPACE_1")"
CMD_ACC2="$(create_instance_cmd "acc2" "$WORKSPACE_2")"

# ตรวจสอบ Terminal Emulator สำหรับรันแบบ Interactive GUI
echo "กำลังเปิด Antigravity แยก 2 หน้าต่าง (Profiles: acc1, acc2)..."

if command -v wt.exe &> /dev/null; then
  # Windows Terminal (ผ่าน WSL หรือ Git Bash)
  wt.exe -w 0 split-pane -d "$WORKSPACE_1" bash -c "$CMD_ACC1; exec bash" \; split-pane -d "$WORKSPACE_2" bash -c "$CMD_ACC2; exec bash"
elif command -v gnome-terminal &> /dev/null; then
  # Linux (GNOME)
  gnome-terminal --title="Antigravity Profile: acc1" -- bash -c "$CMD_ACC1; exec bash"
  gnome-terminal --title="Antigravity Profile: acc2" -- bash -c "$CMD_ACC2; exec bash"
elif command -v osascript &> /dev/null; then
  # macOS (Terminal.app)
  osascript -e "tell application \"Terminal\" to do script \"$CMD_ACC1; exec bash\""
  osascript -e "tell application \"Terminal\" to do script \"$CMD_ACC2; exec bash\""
elif command -v tmux &> /dev/null; then
  # CLI / SSH Fallback (tmux split)
  tmux new-session -d -s agy_multi "bash -c '$CMD_ACC1; exec bash'"
  tmux split-window -h -t agy_multi "bash -c '$CMD_ACC2; exec bash'"
  tmux attach -t agy_multi
else
  echo "คำเตือน: ไม่พบ Terminal Emulator ที่รองรับการแยกหน้าต่างอัตโนมัติ"
  echo "คุณสามารถสั่งรันแต่ละโปรไฟล์แบบ manual ได้ดังนี้:"
  echo "--- Terminal 1 (acc1) ---"
  echo "$CMD_ACC1"
  echo "--- Terminal 2 (acc2) ---"
  echo "$CMD_ACC2"
  exit 1
fi

echo "เปิด 2 Instances สำเร็จเรียบร้อยแล้ว"
