set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

export CLI_BUCKET_NAME="cortex-cli-david"
$ROOT/build/cli.sh
