# Release v0.0.15

## What's Changed
* feat: add 'signature' header to gRPC metadata forwarding by @eugene-twix in https://github.com/tonica-go/tonica/pull/28

## New Features
This release adds support for custom gRPC header forwarding. The framework now allows applications to specify additional HTTP headers that should be forwarded to gRPC metadata, beyond the default set of headers (authorization, traceparent, tracestate, x-request-id).

### Key Changes
- Added `WithCustomGrpcHeaders()` option to configure custom headers for forwarding
- Enhanced header matching logic to support dynamic custom headers
- Improved header key normalization to lowercase for consistent matching

### Technical Details
The changes enable developers to forward custom headers (like 'signature') from HTTP requests to gRPC service metadata, which is useful for authentication, request signing, and other cross-cutting concerns.

## New Contributors
* @eugene-twix made their first contribution in https://github.com/tonica-go/tonica/pull/28

**Full Changelog**: https://github.com/tonica-go/tonica/compare/v0.0.14...v0.0.15

---

## Release Instructions

The commit to be tagged is already on main branch: `7008dd8`

To create this release on GitHub:

1. Create the tag on commit `7008dd8`:
   ```bash
   git checkout main
   git pull
   git tag -a v0.0.15 7008dd8 -m "Release v0.0.15"
   git push origin v0.0.15
   ```

2. Or create the release directly on GitHub:
   - Go to https://github.com/tonica-go/tonica/releases/new
   - Click "Choose a tag" and type `v0.0.15`
   - Click "Create new tag: v0.0.15 on publish"
   - Select target: `main` (commit 7008dd8)
   - Set release title: "v0.0.15"
   - Copy the "What's Changed" and "New Features" sections above into the release description
   - Click "Publish release"

## Verification

The release has been verified:
- ✅ Code builds successfully
- ✅ All tests pass
- ✅ No breaking changes
- ✅ Changes are backward compatible
