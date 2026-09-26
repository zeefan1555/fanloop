declare -a check_names=()
declare -a check_pids=()
declare -a check_logs=()

run_check() {
  local name="$1"
  shift
  local index="${#check_pids[@]}"
  local log="$check_log_root/$name.log"
  printf '[START] %s\n' "$name"
  "$@" >"$log" 2>&1 &
  check_names[$index]="$name"
  check_pids[$index]="$!"
  check_logs[$index]="$log"
}

wait_for_checks() {
  local exit_code=0 index status
  for ((index = 0; index < ${#check_pids[@]}; index++)); do
    if wait "${check_pids[$index]}"; then
      status=PASS
    else
      status=FAIL
      exit_code=1
    fi
    printf '[%s] %s\n' "$status" "${check_names[$index]}"
    cat "${check_logs[$index]}"
  done
  return "$exit_code"
}
