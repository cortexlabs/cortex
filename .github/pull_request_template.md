Closes #<issue ID>

---
Checklist:
- [ ] Run `make test` and `make lint`
- [ ] Test end to end manually (e.g. build/push all images, restart operator, and run `cx refresh`)
- [ ] Update examples
- [ ] Update documentation (add any new files to `summary.md`)
- [ ] Merge to master
- [ ] Cherry-pick into release branches if it's a bugfix
- [ ] Check [gitbook](https://docs.cortex.dev/v/master/) that docs look good and links function properly
- [ ] Alert team if dev environment changed
- [ ] Delete the branch once it's merged
