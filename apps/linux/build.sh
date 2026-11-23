#!/usr/bin/env bash
set -euo pipefail
dir="$1/fidelius-ui"
mkdir -p "$dir"
cc -std=gnu11 -O2 -s apps/linux/gtk.c -o "$dir/gtk" $(pkg-config --cflags --libs gtk4)
c++ -std=c++17 -O2 -s apps/linux/qt.cpp -o "$dir/qt" $(pkg-config --cflags --libs Qt6Widgets)
install -m644 apps/macos/Resources/AppIcon.svg "$dir/io.github.amxv.Fidelius.svg"
install -m644 apps/linux/io.github.amxv.Fidelius.desktop "$dir/"
