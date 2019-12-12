closes #<issue ID>

---

checklist:

- [ ] run `make test` and `make lint`
- [ ] test end to end manually (i.e. build/push all images, restart operator, and run `cortex deploy --refresh`)
- [ ] update the examples
- [ ] update the docs and add any new files to `summary.md`
- [ ] merge to master
- [ ] cherry-pick into release branches if applicable
- [ ] check that the docs are up-to-date in [gitbook](https://www.cortex.dev/v/master)
- [ ] alert the dev team if the dev environment changed
- [ ] delete the branch once it's merged
