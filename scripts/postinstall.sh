#!/bin/bash

systemctl --system daemon-reload >/dev/null || true
systemctl enable qumomf.service >/dev/null || true

deb_systemctl=$(command -v deb-systemd-invoke || echo systemctl)
${deb_systemctl} restart qumomf.service >/dev/null || true
