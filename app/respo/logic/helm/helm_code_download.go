package helm

// Both callers use reusable signed attachment URLs, not one-shot download tokens.
// Only extract into the temporary directory until a complete ZIP is available.
// Prefer curl, retaining BusyBox wget compatibility for existing installer images.
// Do not log the signed URL or use literal percent signs (the values writer strips them).
const codePackageDownloadShell = `show_code_package_error() {
  CODE_PACKAGE_DOWNLOAD_URL="$code_package_url" awk '
    {
      line = $0
      url = ENVIRON["CODE_PACKAGE_DOWNLOAD_URL"]
      output = ""
      while (url != "" && (pos = index(line, url)) > 0) {
        output = output substr(line, 1, pos - 1) "[download URL hidden]"
        line = substr(line, pos + length(url))
      }
      print output line
    }
  ' "$tmp_dir/download-error.log" >&2
}
download_code_package() {
  if [ "$download_tool" = curl ]; then
    curl -fLsS --connect-timeout 30 --speed-limit 1 --speed-time 600 -o "$tmp_zip.part" "$code_package_url"
  else
    wget -q -T 600 -O "$tmp_zip.part" "$code_package_url"
  fi
}
download_tool=wget
if command -v curl >/dev/null 2>&1; then
  download_tool=curl
fi
echo "Code package download tool: $download_tool"
download_attempt=1
while :; do
  echo "Downloading code package (attempt $download_attempt/4)"
  download_started=$(date +"$(printf '\045')s")
  if download_code_package 2>"$tmp_dir/download-error.log"; then
    if unzip -oq "$tmp_zip.part" -d "$package_dir" >"$tmp_dir/download-error.log" 2>&1; then
      mv "$tmp_zip.part" "$tmp_zip"
      break
    else
      download_status=$?
      echo "Code package ZIP validation failed (exit $download_status)" >&2
    fi
  else
    download_status=$?
    echo "Code package download failed (exit $download_status)" >&2
  fi
  show_code_package_error
  echo "Code package attempt $download_attempt failed after $(($(date +"$(printf '\045')s") - download_started)) seconds" >&2
  rm -f "$tmp_zip.part"
  rm -rf "$package_dir"
  if [ "$download_attempt" -ge 4 ]; then
    echo "Code package download failed after 4 attempts; installation stopped" >&2
    exit 1
  fi
  download_delay=$((download_attempt * 5))
  echo "Retrying code package download in $download_delay seconds" >&2
  sleep "$download_delay"
  download_attempt=$((download_attempt + 1))
  mkdir -p "$package_dir"
done
`
