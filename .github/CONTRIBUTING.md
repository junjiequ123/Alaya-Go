## Contributing to Alaya

Interested in contributing? That's awesome! Here are some guidelines to get started quickly and easily:
- [Reporting An Issue](#reporting-an-issue)
    - [Bug Reports](#bug-reports)
    - [Feature Requests](#feature-requests)
    - [Change Requests](#change-requests)
- [Working on Alaya](#working-on-Alaya)
    - [Feature Branches](#feature-branches)
    - [Submitting Pull Requests](#submitting-pull-requests)
    - [Testing and Quality Assurance](#testing-and-quality-assurance)
- [Conduct](#conduct)
- [Contributor License & Acknowledgments](#contributor-license--acknowledgments)
- [References](#references)
- [Developers' Guide](https://devdocs.alaya.network/alaya-devdocs/en/)

## Reporting An Issue

If you're about to raise an issue because you think you've found a problem with Alaya, or you'd like to make a request for a new feature in the codebase, or any other reason… please read this first.

The GitHub issue tracker is the preferred channel for [bug reports](#bug-reports), [feature requests](#feature-requests), and [submitting pull requests](#submitting-pull-requests), but please respect the following restrictions:

* Please do **quick search of existing ones**. If you find similar issue or request already exists, please add your comment to enrich it. Let's avoid having duplicated issues or requests. 

* Please **be civil**. Keep the discussion on topic and respect the opinions of others. See also our [Contributor Code of Conduct](#conduct).

### Bug Reports

A bug is a _demonstrable problem_ that is caused by the code in the repository. Good bug reports are extremely helpful - thank you!

Guidelines for bug reports:

1. **Use the GitHub issue search** &mdash; check if the issue has already been reported or a very similar one exists.

1. **Check if the issue has been fixed** &mdash; look for [closed issues in the current milestone](https://github.com/AlayaNetwork/Alaya-Go/issues?q=is%3Aissue+is%3Aclosed) or try to reproduce it using the latest `develop` branch.

A good bug report shouldn't leave others needing to chase you up for more information. Be sure to include the enough details of your environment, current status, test cases, and ideally steps to re-produce the failure.

[Report a bug](https://github.com/AlayaNetwork/Alaya-Go/issues/new?assignees=&labels=bug&template=bug.md&title=)

### Feature Requests

Feature requests are welcome. Before you submit one be sure to have:

1. **Use the GitHub search** and check if a similar feature request exists or not, if yes, support it with your comment.
1. Take a moment to re-think about whether your idea fits with the scope and aims of the project.
1. Remember, it's up to *you* to make a strong case to get the community leaders understand and support your feature request. Please provide as much detail and context as possible, this means explaining the typical use cases and why it is likely to be common.

### Change Requests

Change requests cover both architectural and functional changes to how Alaya works. If you have an idea for a refactor, or an improvement to a feature, or adding/replacing of dependencies, etc. - please be sure to:

1. **Use the GitHub search** and check someone else didn't get there first
1. Take a moment to think about the best way to make a case for, and explain why and what you're thinking. Are you sure this shouldn't be a [bug report](#bug-reports) or a [feature request](#feature-requests)?  Is this the only way to do so or there are many other alternatives? What's the context? What problem are you trying to solve? Why your suggestion is better than existing logic?

## Working on Alaya

Code contributions are welcome and encouraged! If you are looking for a good place to start, check out the [good first issue](https://github.com/AlayaNetwork/Alaya-Go/labels/good%20first%20issue) label in GitHub issues.

Also, please follow these guidelines when submitting code:

### Feature Branches

To get it out of the way:

- **[feature/xxx](https://github.com/AlayaNetwork/Alaya-Go/tree/feature/bump-version-to-0.16.0)** is the development for new version feature branch. All work on the next version release happens here so you should generally branch off `feature/xxx`. All feature branches are unstable and NOT ready for production, please do **NOT** use any feature branch for a production.
- **[develop](https://github.com/AlayaNetwork/Alaya-Go/tree/develop)** is the development branch. Bug fixes of the current version can be submitted to this branch. Do **NOT** use this branch for a production.
- **[master](https://github.com/AlayaNetwork/Alaya-Go/tree/master)** contains the latest release of Alaya. This is the branch could be used in production. Do **NOT** work on this branch for any Alaya updates. 

### Submitting Pull Requests (PR)

Pull requests are awesome. Raising PR means you are asking to merge your changes into target branch, and closing an existing issue. If you are raising a PR without any open issue related, please [raise one](#reporting-an-issue) in advance, especially if you're fixing a bug. This makes it more likely that there will be enough information available for your PR to be properly tested and merged. 

### Testing and Quality Assurance

Never underestimate just how useful quality assurance is. If you're looking to get involved with the code base and don't know where to start, checking out and testing a pull request is one of the most useful things you could do.

Essentially, [check out the latest develop branch](#working-on-Alaya), take it for a spin, and if you find anything odd, please follow the [bug report guidelines](#bug-reports) and let us know!

## Conduct

While contributing, please be respectful and constructive, so that participation in our project is a positive experience for everyone.

Examples of behavior that contributes to creating a positive environment include:
- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

Examples of unacceptable behavior include:
- The use of sexualized language or imagery and unwelcome sexual attention or advances
- Trolling, insulting/derogatory comments, and personal or political attacks
- Public or private harassment
- Publishing others’ private information, such as a physical or electronic address, without explicit permission
- Other conduct which could reasonably be considered inappropriate in a professional setting

## Contributor License & Acknowledgments

Whenever you make a contribution to this project, you license your contribution under the same terms as set out in [LICENSE](./COPYING), and you represent and warrant that you have the right to license your contribution under those terms.  Whenever you make a contribution to this project, you also certify in the terms of the Developer’s Certificate of Origin set out below:

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
1 Letterman Drive
Suite D4700
San Francisco, CA, 94129

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.


Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

## References
* Overall CONTRIB adapted from https://github.com/mathjax/MathJax/blob/master/CONTRIBUTING.md
* Conduct section adapted from the Contributor Covenant, version 1.4, available at https://www.contributor-covenant.org/version/1/4/code-of-conduct.html
