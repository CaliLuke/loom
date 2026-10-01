#!/bin/zsh
set -euo pipefail
readonly source_root=/private/tmp/loom572-full934-candidate-freeze4/common-source
readonly result_root=/tmp/loom572-full934-sharded-results
readonly overlay=/tmp/loom572-full934-diagnostic-overlay.json
readonly driver=/tmp/loom572-full934-sharded-driver.go
readonly support=/tmp/loom572-full934-support.go
readonly diagnostic=/tmp/loom571-full934-diagnostic-continuation.go
readonly manifest=/tmp/loom572-full934-manifest.json
readonly start_shard=${LOOM571_START_SHARD:-0}
readonly minimum_kib=$((16 * 1024 * 1024))
[[ $(shasum -a 256 "$driver" | awk '{print $1}') == 8c6ac9b4d31d1c05f0d3bb6023fa93f70678b77274940823be7c5b2f977c67d3 ]]
[[ $(shasum -a 256 "$support" | awk '{print $1}') == 1ad78f0ff07b4447353cb4300c6326cbd26b0102bc6c476c9759acc74164768f ]]
[[ $(shasum -a 256 "$diagnostic" | awk '{print $1}') == 520d5911251a099afe2f8526e35a59f515f582bccf0dc0d03331b75c1c1d420a ]]
[[ $(shasum -a 256 "$overlay" | awk '{print $1}') == a39f9227e50c439730496794cd6bcd3f57678f0f8a9e952a4e7bbcf1484486db ]]
[[ $(shasum -a 256 "$manifest" | awk '{print $1}') == e2e92e9132c57ed060c712d45f9c7e8b2814a8e3cf876a399a1ab71bccdbae8b ]]
(( start_shard >= 0 && start_shard <= 58 ))
(( $(df -k /private/tmp | awk 'NR == 2 {print $4}') >= minimum_kib ))
[[ ! -e "$result_root" || -d "$result_root" ]]
readonly discovery_log=/tmp/loom572-full934-freeze4-discovery.log
[[ ! -e "$discovery_log" ]]
(cd "$source_root" && go test -overlay="$overlay" ./internal/valuecontract -list '^TestLoom571Shard(Generation|RetainedPilot|DiagnosticAcquisition)$') 2>&1 | tee "$discovery_log"
grep -Fxq 'TestLoom571ShardGeneration' "$discovery_log"
grep -Fxq 'TestLoom571ShardRetainedPilot' "$discovery_log"
grep -Fxq 'TestLoom571ShardDiagnosticAcquisition' "$discovery_log"
for ((prior = 0; prior < start_shard; prior++)); do
  tag=$(printf '%02d' "$prior")
  receipt="$result_root/logs/shard-$tag-receipt.txt"
  quarantine="$result_root/logs/shard-$tag-quarantine-receipt.txt"
  if [[ -f "$receipt" ]]; then
    [[ ! -e "$quarantine" ]]
    expected=$(shasum -a 256 "$result_root/shards/$tag.tar.gz" "$result_root/shards/$tag.json" "$result_root/logs/shard-$tag-generation.log" "$result_root/logs/shard-$tag-retained.log")
    [[ $(cat "$receipt") == "$expected" ]]
    continue
  fi
  [[ -f "$quarantine" ]]
  [[ ! -e "$result_root/logs/shard-$tag-generation.log" ]]
  [[ ! -e "$result_root/logs/shard-$tag-retained.log" ]]
  expected=$(shasum -a 256 "$result_root/shards/$tag.tar.gz" "$result_root/shards/$tag.json" "$result_root/logs/shard-$tag-generation-failed.log" "$result_root/logs/shard-$tag-diagnostic.log" "$result_root/logs/shard-$tag-diagnostic.json")
  [[ $(cat "$quarantine") == "$expected" ]]
done
for ((index = start_shard; index <= 58; index++)); do
  (( $(df -k /private/tmp | awk 'NR == 2 {print $4}') >= minimum_kib ))
  tag=$(printf '%02d' "$index")
  generation_tmp="/tmp/loom572-full934-shard-$tag-generation-inflight.log"
  retained_tmp="/tmp/loom572-full934-shard-$tag-retained-inflight.log"
  diagnostic_tmp="/tmp/loom572-full934-shard-$tag-diagnostic-inflight.log"
  [[ ! -e "$generation_tmp" && ! -e "$retained_tmp" && ! -e "$diagnostic_tmp" ]]
  [[ ! -e "$result_root/shards/$tag.tar.gz" && ! -e "$result_root/shards/$tag.json" ]]
  [[ ! -e "$result_root/expanded/$tag" && ! -e "$result_root/rehydrated/$tag" ]]
  [[ ! -e "$result_root/logs/shard-$tag-generation.log" && ! -e "$result_root/logs/shard-$tag-retained.log" ]]
  [[ ! -e "$result_root/logs/shard-$tag-generation-failed.log" && ! -e "$result_root/logs/shard-$tag-diagnostic.log" && ! -e "$result_root/logs/shard-$tag-diagnostic.json" ]]
  [[ ! -e "$result_root/logs/shard-$tag-receipt.txt" && ! -e "$result_root/logs/shard-$tag-quarantine-receipt.txt" ]]
  set +e
  (cd "$source_root" && LOOM571_SHARD_INDEX=$index go test -overlay="$overlay" ./internal/valuecontract -run '^TestLoom571ShardGeneration$' -parallel=2 -count=1 -v -timeout=3h) 2>&1 | tee "$generation_tmp"
  generation_statuses=("${pipestatus[@]}")
  generation_command_exit=${generation_statuses[1]}
  generation_tee_exit=${generation_statuses[2]}
  set -e
  (( generation_tee_exit == 0 ))
  grep -q '=== RUN   TestLoom571ShardGeneration' "$generation_tmp"
  [[ -f "$result_root/shards/$tag.tar.gz" && -f "$result_root/shards/$tag.json" ]]
  mkdir -p "$result_root/logs"
  if (( generation_command_exit != 0 )); then
    grep -q -- '^--- FAIL: TestLoom571ShardGeneration' "$generation_tmp"
    mv "$generation_tmp" "$result_root/logs/shard-$tag-generation-failed.log"
    (cd "$source_root" && LOOM571_SHARD_INDEX=$index go test -overlay="$overlay" ./internal/valuecontract -run '^TestLoom571ShardDiagnosticAcquisition$' -parallel=2 -count=1 -v -timeout=30m) 2>&1 | tee "$diagnostic_tmp"
    grep -q '=== RUN   TestLoom571ShardDiagnosticAcquisition' "$diagnostic_tmp"
    grep -q -- '--- PASS: TestLoom571ShardDiagnosticAcquisition' "$diagnostic_tmp"
    mv "$diagnostic_tmp" "$result_root/logs/shard-$tag-diagnostic.log"
    [[ -f "$result_root/logs/shard-$tag-diagnostic.json" ]]
    shasum -a 256 "$result_root/shards/$tag.tar.gz" "$result_root/shards/$tag.json" "$result_root/logs/shard-$tag-generation-failed.log" "$result_root/logs/shard-$tag-diagnostic.log" "$result_root/logs/shard-$tag-diagnostic.json" > "$result_root/logs/shard-$tag-quarantine-receipt.txt"
  else
    grep -q -- '^--- PASS: TestLoom571ShardGeneration' "$generation_tmp"
    mv "$generation_tmp" "$result_root/logs/shard-$tag-generation.log"
    (cd "$source_root" && LOOM571_SHARD_INDEX=$index go test -overlay="$overlay" ./internal/valuecontract -run '^TestLoom571ShardRetainedPilot$' -parallel=2 -count=1 -v -timeout=30m) 2>&1 | tee "$retained_tmp"
    grep -q '=== RUN   TestLoom571ShardRetainedPilot' "$retained_tmp"
    grep -q -- '--- PASS: TestLoom571ShardRetainedPilot' "$retained_tmp"
    mv "$retained_tmp" "$result_root/logs/shard-$tag-retained.log"
    shasum -a 256 "$result_root/shards/$tag.tar.gz" "$result_root/shards/$tag.json" "$result_root/logs/shard-$tag-generation.log" "$result_root/logs/shard-$tag-retained.log" > "$result_root/logs/shard-$tag-receipt.txt"
  fi
done
