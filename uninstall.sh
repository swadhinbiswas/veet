#!/usr/bin/env bash
# ==============================================================================
#  VEET Uninstaller
# ==============================================================================

set -e

BOLD='\033[1m'
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

printf "\n${CYAN}${BOLD}VEET Uninstaller${NC}\n\n"

# Remove binaries
REMOVED=0
for BIN in "$HOME/.local/bin/veet" "/usr/local/bin/veet" "/usr/bin/veet" "$HOME/.local/bin/unixpurge" "/usr/local/bin/unixpurge"; do
    if [ -f "$BIN" ]; then
        if [ -w "$BIN" ]; then
            rm -f "$BIN"
            printf "${GREEN}✓${NC} Removed: %s\n" "$BIN"
            REMOVED=$((REMOVED + 1))
        else
            printf "${YELLOW}🔒 Requires sudo to remove: %s${NC}\n" "$BIN"
            sudo rm -f "$BIN"
            printf "${GREEN}✓${NC} Removed: %s\n" "$BIN"
            REMOVED=$((REMOVED + 1))
        fi
    fi
done

# Remove completions
rm -f "$HOME/.local/share/bash-completion/completions/veet" "$HOME/.local/share/bash-completion/completions/unixpurge" 2>/dev/null || true
rm -f "$HOME/.local/share/zsh/site-functions/_veet" "$HOME/.local/share/zsh/site-functions/_unixpurge" 2>/dev/null || true
rm -f "$HOME/.config/fish/completions/veet.fish" "$HOME/.config/fish/completions/unixpurge.fish" 2>/dev/null || true
printf "${GREEN}✓${NC} Removed shell completions\n"

# Optionally remove data / history
if [ -d "$HOME/.local/share/veet" ] || [ -d "$HOME/.config/veet" ] || [ -d "$HOME/.local/share/unixpurge" ]; then
    printf "\n${YELLOW}Do you also want to remove audit logs and configurations? (y/N):${NC} "
    read -r CONFIRM < /dev/tty || CONFIRM="n"
    if [[ "$CONFIRM" =~ ^[Yy]$ ]]; then
        rm -rf "$HOME/.local/share/veet" "$HOME/.config/veet" "$HOME/.local/share/unixpurge" "$HOME/.config/unixpurge"
        printf "${GREEN}✓${NC} Removed logs and configuration files.\n"
    else
        printf "${CYAN}•${NC} Kept configuration and history logs.\n"
    fi
fi

printf "\n${GREEN}${BOLD}VEET has been uninstalled.${NC}\n\n"
