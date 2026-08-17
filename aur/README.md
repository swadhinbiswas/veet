# Publishing VEET to the Arch User Repository (AUR)

This directory contains pre-configured Arch Linux packaging files:

- **`aur/veet/`**: Stable release package (builds from official GitHub tagged releases)
- **`aur/veet-git/`**: VCS development package (builds directly from the latest `main` branch)

---

## Prerequisites

1. An AUR account on [aur.archlinux.org](https://aur.archlinux.org).
2. Your SSH Public Key added to [AUR Account Settings](https://aur.archlinux.org/account).
3. `base-devel` and `git` installed on your Arch system:
   ```bash
   sudo pacman -S --needed base-devel git
   ```

---

## Step 1: Verify AUR SSH Access

Test your SSH connection to the Arch User Repository:

```bash
ssh -T aur@aur.archlinux.org
```

Expected output:
```text
Interactive shell is disabled.
Welcome to AUR, <your_username>!
```

---

## Step 2: Publish `veet` (Stable Package)

### 1. Clone the AUR repository

```bash
git clone ssh://aur@aur.archlinux.org/veet.git /tmp/aur-veet
```

### 2. Copy the PKGBUILD & .SRCINFO

```bash
cp aur/veet/PKGBUILD aur/veet/.SRCINFO /tmp/aur-veet/
cd /tmp/aur-veet
```

### 3. Test Build Locally

```bash
makepkg -si
```

### 4. Commit and Push to AUR

```bash
git add PKGBUILD .SRCINFO
git commit -m "feat: initial release v1.0.1"
git push origin master
```

---

## Step 3: Publish `veet-git` (VCS Git Package, Optional)

```bash
git clone ssh://aur@aur.archlinux.org/veet-git.git /tmp/aur-veet-git
cp aur/veet-git/PKGBUILD aur/veet-git/.SRCINFO /tmp/aur-veet-git/
cd /tmp/aur-veet-git
makepkg -si
git add PKGBUILD .SRCINFO
git commit -m "feat: initial veet-git release"
git push origin master
```

---

## Updating Future Releases

Whenever a new version of VEET is released:

1. Update `pkgver` in `aur/veet/PKGBUILD`.
2. Update the sha256 checksum:
   ```bash
   updpkgsums
   ```
3. Regenerate `.SRCINFO`:
   ```bash
   makepkg --printsrcinfo > .SRCINFO
   ```
4. Commit and push to `origin master`.
