#!/usr/bin/env bash
# ==============================================================================
#  VEET Installer
#  Universal Linux App Uninstaller & Deep-Clean Residual Purger
# ==============================================================================

set -e

# Colors
BOLD='\033[1m'
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

printf "\n${CYAN}${BOLD}╔═══════════════════════════════════════════════╗${NC}\n"
printf "${CYAN}${BOLD}║              VEET Auto-Installer              ║${NC}\n"
printf "${CYAN}${BOLD}║      Universal Linux App Uninstaller          ║${NC}\n"
printf "${CYAN}${BOLD}╚═══════════════════════════════════════════════╝${NC}\n\n"

# 1. Determine destination directory
if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
fi

mkdir -p "$INSTALL_DIR"

# 2. Locate Go compiler
GO_CMD=""
if command -v go >/dev/null 2>&1; then
    GO_CMD="go"
elif [ -x "$HOME/.local/go/bin/go" ]; then
    GO_CMD="$HOME/.local/go/bin/go"
elif [ -x "/usr/local/go/bin/go" ]; then
    GO_CMD="/usr/local/go/bin/go"
fi

if [ -z "$GO_CMD" ]; then
    printf "${RED}✗ Error:${NC} Go compiler (Go 1.24+) not found.\n"
    printf "Please install Go from ${CYAN}https://go.dev/dl/${NC} and run this installer again.\n\n"
    exit 1
fi

GO_VERSION=$($GO_CMD version | awk '{print $3}')
printf "${GREEN}✓${NC} Found Go compiler: ${BOLD}%s${NC} (%s)\n" "$GO_CMD" "$GO_VERSION"

# 3. Determine build directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_BUILD=""

if [ -f "$SCRIPT_DIR/main.go" ] && [ -f "$SCRIPT_DIR/go.mod" ]; then
    SRC_DIR="$SCRIPT_DIR"
    printf "${GREEN}✓${NC} Installing from local source: ${BOLD}%s${NC}\n" "$SRC_DIR"
else
    printf "${YELLOW}•${NC} Fetching latest VEET repository...\n"
    TMP_BUILD=$(mktemp -d)
    git clone --depth 1 https://github.com/swadhinbiswas/veet.git "$TMP_BUILD"
    SRC_DIR="$TMP_BUILD"
fi

# Cleanup on exit if temp dir was created
cleanup() {
    if [ -n "$TMP_BUILD" ] && [ -d "$TMP_BUILD" ]; then
        rm -rf "$TMP_BUILD"
    fi
}
trap cleanup EXIT

# 4. Build single binary
printf "${YELLOW}•${NC} Compiling veet single binary...\n"
cd "$SRC_DIR"
"$GO_CMD" build -ldflags="-s -w" -o "$INSTALL_DIR/veet" .
chmod +x "$INSTALL_DIR/veet"

printf "${GREEN}✓${NC} Installed binary to: ${BOLD}%s/veet${NC}\n" "$INSTALL_DIR"

# 5. Shell autocompletions
printf "${YELLOW}•${NC} Configuring shell autocompletions...\n"

# Bash completion
if [ -d "$HOME/.local/share/bash-completion/completions" ] || mkdir -p "$HOME/.local/share/bash-completion/completions" 2>/dev/null; then
    "$INSTALL_DIR/veet" completion bash > "$HOME/.local/share/bash-completion/completions/veet" 2>/dev/null || true
fi

# Zsh completion
if [ -d "$HOME/.local/share/zsh/site-functions" ] || mkdir -p "$HOME/.local/share/zsh/site-functions" 2>/dev/null; then
    "$INSTALL_DIR/veet" completion zsh > "$HOME/.local/share/zsh/site-functions/_veet" 2>/dev/null || true
fi

# Fish completion
if [ -d "$HOME/.config/fish/completions" ] || mkdir -p "$HOME/.config/fish/completions" 2>/dev/null; then
    "$INSTALL_DIR/veet" completion fish > "$HOME/.config/fish/completions/veet.fish" 2>/dev/null || true
fi

printf "${GREEN}✓${NC} Shell completions generated.\n"

# 6. Auto-install Nerd Font Symbols if missing
printf "${YELLOW}•${NC} Checking Nerd Font icon support...\n"
HAS_NERD=false
if command -v fc-list >/dev/null 2>&1; then
    if fc-list : family | grep -qi "Nerd Font"; then
        HAS_NERD=true
        printf "${GREEN}✓${NC} Nerd Font detected in fontconfig.\n"
    fi
fi

if [ "$HAS_NERD" = false ]; then
    if [ "$(id -u)" -eq 0 ]; then
        FONT_DIR="/usr/local/share/fonts/NerdFonts"
    else
        FONT_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/fonts/NerdFonts"
    fi
    mkdir -p "$FONT_DIR"
    FONT_FILE="$FONT_DIR/SymbolsNerdFont-Regular.ttf"
    FONT_MONO="$FONT_DIR/SymbolsNerdFontMono-Regular.ttf"

    if [ ! -f "$FONT_FILE" ]; then
        printf "${YELLOW}•${NC} Downloading official Nerd Font Symbols into ${BOLD}%s${NC}...\n" "$FONT_DIR"
        FONT_URL="https://github.com/ryanoasis/nerd-fonts/raw/HEAD/patched-fonts/NerdFontsSymbolsOnly/SymbolsNerdFont-Regular.ttf"
        FONT_MONO_URL="https://github.com/ryanoasis/nerd-fonts/raw/HEAD/patched-fonts/NerdFontsSymbolsOnly/SymbolsNerdFontMono-Regular.ttf"
        
        DOWNLOADED=false
        if command -v curl >/dev/null 2>&1; then
            if curl -fLo "$FONT_FILE" "$FONT_URL" 2>/dev/null && curl -fLo "$FONT_MONO" "$FONT_MONO_URL" 2>/dev/null; then
                DOWNLOADED=true
            fi
        elif command -v wget >/dev/null 2>&1; then
            if wget -qO "$FONT_FILE" "$FONT_URL" 2>/dev/null && wget -qO "$FONT_MONO" "$FONT_MONO_URL" 2>/dev/null; then
                DOWNLOADED=true
            fi
        fi

        if [ "$DOWNLOADED" = true ]; then
            if command -v fc-cache >/dev/null 2>&1; then
                fc-cache -f "$FONT_DIR" 2>/dev/null || true
            fi
            printf "${GREEN}✓${NC} Installed Nerd Font Symbols fallback font.\n"
        else
            printf "${YELLOW}• Note:${NC} Could not auto-download font. VEET will use universal Unicode fallback (${CYAN}icons: auto${NC}).\n"
        fi
    else
        printf "${GREEN}✓${NC} Nerd Font Symbols already present in %s.\n" "$FONT_DIR"
    fi
fi

# 7. Verify PATH
IN_PATH=false
case ":$PATH:" in
    *":$INSTALL_DIR:"*) IN_PATH=true ;;
esac

printf "\n${GREEN}${BOLD}🎉 Installation Complete!${NC}\n\n"

if [ "$IN_PATH" = false ]; then
    printf "${YELLOW}⚠ Notice:${NC} ${BOLD}%s${NC} is not currently in your \$PATH.\n" "$INSTALL_DIR"
    printf "Add it by running the following command (or add to your ~/.bashrc / ~/.zshrc):\n\n"
    printf "    ${CYAN}export PATH=\"\$PATH:%s\"${NC}\n\n" "$INSTALL_DIR"
fi

printf "To launch VEET:\n"
printf "    ${CYAN}veet${NC}         # Launch interactive TUI\n"
printf "    ${CYAN}veet scan${NC}    # List installed apps across all sources\n"
printf "    ${CYAN}veet --help${NC}  # View CLI commands and flags\n\n"
