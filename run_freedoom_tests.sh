#!/bin/bash

set -euo pipefail

WAD_DIR=$(realpath "${1:?usage: run_freedoom_tests.sh WAD_DIRECTORY OUTPUT_DIRECTORY}")
OUTPUT_DIR=$(realpath -m "${2:?provide an output directory}")
ROOT_DIR=$(cd "$(dirname "$0")" && pwd)
BUILD_DIR=$(mktemp -d)
trap 'rm -rf "$BUILD_DIR"' EXIT

(cd "$WAD_DIR" && sha256sum --check <<'SUMS'
7323bcc168c5a45ff10749b339960e98314740a734c30d4b9f3337001f9e703d  freedoom1.wad
a8772e088847032510d97ba2312406a6998f21cbab44d4ff10696faa9c0ecd4b  freedoom2.wad
SUMS
)

cd "$ROOT_DIR"
go test doom.go safe_data.go frontend.go safe_data_test.go frontend_test.go
go test -c -o "$BUILD_DIR/replay.test" doom.go safe_data.go frontend.go freedoom_test.go

run_replay() {
	local phase=$1 map=$2 scenario=$3
	local destination="$OUTPUT_DIR/phase$phase-${map// /-}-$scenario"
	mkdir -p "$destination"
	(cd "$destination" && GORE_REPLAY_WAD="$WAD_DIR/freedoom$phase.wad" \
		GORE_REPLAY_MAP="$map" GORE_REPLAY_FRAMES=120 \
		GORE_REPLAY_OUTPUT="$destination/frames.json" \
		"$BUILD_DIR/replay.test" -test.run '^TestFreedoomReplay$' -test.v > run.log 2>&1)
	echo "Passed phase $phase map $map ($scenario)"
}

for episode in {1..4}; do
	for map in {1..9}; do
		run_replay 1 "$episode $map" idle
	done
done
for map in {1..32}; do
	run_replay 2 "$map" idle
done
for phase in 1 2; do
	map=1
	if [ "$phase" = 1 ]; then map='1 1'; fi
	GORE_REPLAY_MOVE=1 GORE_REPLAY_SAVE=1 run_replay "$phase" "$map" actions
	destination="$OUTPUT_DIR/phase$phase-${map// /-}"
	mkdir -p "$destination-load/.savegame"
	cp "$destination-actions/.savegame/dgsave0.dsg" "$destination-load/.savegame/dgsave0.dsg"
	GORE_REPLAY_LOAD_SLOT=0 run_replay "$phase" "$map" load
done
