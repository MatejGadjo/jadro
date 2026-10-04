#!/usr/bin/env bash
# Downloads the sentence-embedding model and ONNX Runtime into backend/models/.
# Every file is pinned to an exact version and verified by SHA-256.
#
#   ./scripts/download-models.sh            # ONNX Runtime for this machine's CPU
#   ORT_ARCH=x64 ./scripts/download-models.sh   # force x64 (or aarch64)
#
# Model: sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2 (Apache-2.0)
# Runtime: microsoft/onnxruntime 1.23.2 (MIT)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODELS_DIR="${MODELS_DIR:-$ROOT/backend/models}"

HF_REPO="sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2"
HF_REV="e8f8c211226b894fcb81acc59f3b34ba3efd5f42"
MODEL_DIR="$MODELS_DIR/paraphrase-multilingual-MiniLM-L12-v2"
MODEL_SHA256="10f7a088420252b26caf819236ca2c9d2987afd0fc06fec7553b542a5655a05a"
TOKENIZER_SHA256="2c3387be76557bd40970cec13153b3bbf80407865484b209e655e5e4729076b8"

ORT_VERSION="1.23.2"  # must match github.com/yalue/onnxruntime_go in backend/go.mod (v1.25.x)
ORT_X64_SHA256="1fa4dcaef22f6f7d5cd81b28c2800414350c10116f5fdd46a2160082551c5f9b"
ORT_AARCH64_SHA256="7c63c73560ed76b1fac6cff8204ffe34fe180e70d6582b5332ec094810241e5c"

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1
  else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

# fetch URL DEST SHA256 — skips the download when DEST already matches.
fetch() {
  local url="$1" dest="$2" want="$3"
  if [[ -f "$dest" && "$(sha256 "$dest")" == "$want" ]]; then
    echo "✓ $(basename "$dest") already present"
    return
  fi
  echo "↓ $(basename "$dest")"
  curl -fL --retry 3 --progress-bar -o "$dest.part" "$url"
  local got
  got="$(sha256 "$dest.part")"
  if [[ "$got" != "$want" ]]; then
    rm -f "$dest.part"
    echo "✗ checksum mismatch for $(basename "$dest"): got $got, want $want" >&2
    exit 1
  fi
  mv "$dest.part" "$dest"
}

# ONNX Runtime must match the CPU of the *container*, which on Docker Desktop
# is the host CPU (Apple Silicon → aarch64).
if [[ -z "${ORT_ARCH:-}" ]]; then
  case "$(uname -m)" in
    x86_64|amd64) ORT_ARCH=x64 ;;
    arm64|aarch64) ORT_ARCH=aarch64 ;;
    *) echo "Unsupported CPU $(uname -m); set ORT_ARCH=x64 or aarch64" >&2; exit 1 ;;
  esac
fi
case "$ORT_ARCH" in
  x64) ORT_SHA256="$ORT_X64_SHA256" ;;
  aarch64) ORT_SHA256="$ORT_AARCH64_SHA256" ;;
  *) echo "ORT_ARCH must be x64 or aarch64" >&2; exit 1 ;;
esac

mkdir -p "$MODEL_DIR"
fetch "https://huggingface.co/$HF_REPO/resolve/$HF_REV/onnx/model.onnx" "$MODEL_DIR/model.onnx" "$MODEL_SHA256"
fetch "https://huggingface.co/$HF_REPO/resolve/$HF_REV/tokenizer.json" "$MODEL_DIR/tokenizer.json" "$TOKENIZER_SHA256"

ORT_NAME="onnxruntime-linux-$ORT_ARCH-$ORT_VERSION"
if [[ -f "$MODELS_DIR/$ORT_NAME/lib/libonnxruntime.so" ]]; then
  echo "✓ $ORT_NAME already present"
else
  TGZ="$MODELS_DIR/$ORT_NAME.tgz"
  fetch "https://github.com/microsoft/onnxruntime/releases/download/v$ORT_VERSION/$ORT_NAME.tgz" "$TGZ" "$ORT_SHA256"
  tar -xzf "$TGZ" -C "$MODELS_DIR"
  rm -f "$TGZ"
fi

echo
echo "Models ready in $MODELS_DIR"
if [[ "$ORT_ARCH" != "x64" ]]; then
  echo "Add this to your .env so Docker loads the $ORT_ARCH runtime:"
  echo "  ORT_ARCH=$ORT_ARCH"
fi
