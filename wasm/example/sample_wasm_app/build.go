package sample_wasm_app

//go:generate bash -c "GOARCH=wasm; GOOS=js; go build -o ${OUT_DIR}/main.wasm main.go"
