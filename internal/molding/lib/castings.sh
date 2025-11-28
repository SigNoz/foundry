#!/bin/bash
# castings/foundry.sh

TARGET=$1
ACTION=$2

if [ -z "$TARGET" ] || [ -z "$ACTION" ]; then
    echo "Usage: ./castings.sh <path/to/casting> {forge|cast|scrap}"
    exit 1
fi

CASTING_DIR="$(cd "$(dirname "$0")" && pwd)/$TARGET"

if [ ! -f "$CASTING_DIR/setup.sh" ]; then
    echo "❌ Error: No setup.sh found in $CASTING_DIR"
    exit 1
fi

echo "========================================"
echo "🏭 FOUNDRY: $TARGET"
echo "🔧 ACTION:  $ACTION"
echo "========================================"

# Execute inside the directory so relative paths work
cd "$CASTING_DIR" && ./setup.sh "$ACTION"