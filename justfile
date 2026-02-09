set shell := ["bash", "-cu"]

go_version := "1.25.0"

product := ""
profiles := ""
workspace := ""
repo_index := ""
output := "out"
repo_dir := "out/repo"
sbom := "true"
repo_backend := "file"
debs_dir := "out/debs"
gpg_key := ""
proget_endpoint := ""
proget_feed := ""
proget_component := "main"
proget_user := ""
proget_api_key := ""
pip_index_url := ""
internal_deb_dir := ""
internal_src := ""
target_ubuntu := ""
snapshot_id := ""

# Build and launch the devcontainer for interactive development
dev:
  @scripts/dev.sh

# Install development dependencies on the host (Arch, Debian, Fedora)
deps:
  @scripts/deps.sh {{go_version}}

default: validate

validate:
  @bash -c '[[ -n "{{product}}" ]] || { echo "product is required"; exit 1; }'
  @bash -c 'cmd="avular-packages validate --product {{product}}"; if [[ -n "{{profiles}}" ]]; then cmd="$cmd --profile {{profiles}}"; fi; eval "$cmd"'

resolve:
  @bash -c '[[ -n "{{product}}" ]] || { echo "product is required"; exit 1; }'
  @bash -c '[[ -n "{{repo_index}}" ]] || { echo "repo_index is required"; exit 1; }'
  @bash -c '[[ -n "{{target_ubuntu}}" ]] || { echo "target_ubuntu is required"; exit 1; }'
  @bash -c 'cmd="avular-packages resolve --product {{product}} --repo-index {{repo_index}} --output {{output}} --target-ubuntu {{target_ubuntu}}"; \
    if [[ -n "{{profiles}}" ]]; then cmd="$cmd --profile {{profiles}}"; fi; \
    if [[ -n "{{workspace}}" ]]; then cmd="$cmd --workspace {{workspace}}"; fi; \
    if [[ -n "{{snapshot_id}}" ]]; then cmd="$cmd --snapshot-id {{snapshot_id}}"; fi; \
    eval "$cmd"'

lock:
  @bash -c '[[ -n "{{product}}" ]] || { echo "product is required"; exit 1; }'
  @bash -c '[[ -n "{{repo_index}}" ]] || { echo "repo_index is required"; exit 1; }'
  @bash -c '[[ -n "{{target_ubuntu}}" ]] || { echo "target_ubuntu is required"; exit 1; }'
  @bash -c 'cmd="avular-packages lock --product {{product}} --repo-index {{repo_index}} --output {{output}} --target-ubuntu {{target_ubuntu}}"; \
    if [[ -n "{{profiles}}" ]]; then cmd="$cmd --profile {{profiles}}"; fi; \
    if [[ -n "{{workspace}}" ]]; then cmd="$cmd --workspace {{workspace}}"; fi; \
    if [[ -n "{{snapshot_id}}" ]]; then cmd="$cmd --snapshot-id {{snapshot_id}}"; fi; \
    eval "$cmd"'

build:
  @bash -c '[[ -n "{{product}}" ]] || { echo "product is required"; exit 1; }'
  @bash -c '[[ -n "{{repo_index}}" ]] || { echo "repo_index is required"; exit 1; }'
  @bash -c '[[ -n "{{target_ubuntu}}" ]] || { echo "target_ubuntu is required"; exit 1; }'
  @bash -c 'cmd="avular-packages build --product {{product}} --repo-index {{repo_index}} --output {{output}} --target-ubuntu {{target_ubuntu}} --debs-dir {{debs_dir}}"; \
    if [[ -n "{{profiles}}" ]]; then cmd="$cmd --profile {{profiles}}"; fi; \
    if [[ -n "{{workspace}}" ]]; then cmd="$cmd --workspace {{workspace}}"; fi; \
    if [[ -n "{{pip_index_url}}" ]]; then cmd="$cmd --pip-index-url {{pip_index_url}}"; fi; \
    if [[ -n "{{internal_deb_dir}}" ]]; then cmd="$cmd --internal-deb-dir {{internal_deb_dir}}"; fi; \
    if [[ -n "{{internal_src}}" ]]; then cmd="$cmd --internal-src {{internal_src}}"; fi; \
    eval "$cmd"'

publish:
  @bash -c 'cmd="avular-packages publish --output {{output}} --repo-dir {{repo_dir}} --sbom {{sbom}} --repo-backend {{repo_backend}} --debs-dir {{debs_dir}} --gpg-key {{gpg_key}}"; \
    if [[ -n "{{proget_endpoint}}" ]]; then cmd="$cmd --proget-endpoint {{proget_endpoint}}"; fi; \
    if [[ -n "{{proget_feed}}" ]]; then cmd="$cmd --proget-feed {{proget_feed}}"; fi; \
    if [[ -n "{{proget_component}}" ]]; then cmd="$cmd --proget-component {{proget_component}}"; fi; \
    if [[ -n "{{proget_user}}" ]]; then cmd="$cmd --proget-user {{proget_user}}"; fi; \
    if [[ -n "{{proget_api_key}}" ]]; then cmd="$cmd --proget-api-key {{proget_api_key}}"; fi; \
    eval "$cmd"'

inspect:
  @avular-packages inspect --output {{output}}

clean:
  @rm -rf "{{output}}"
