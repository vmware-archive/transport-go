# Contribution process for developers

If you plan on contributing code to Transport, please follow this step-by-step
process to reduce the risk of significant changes being requested when you
submit your pull request. We'll work with you on every step and try to be as
responsive as possible.

## Implementation on a topic branch

### Prerequisites

First, make sure you:

- Read our [Developer Certificate of Origin](https://cla.vmware.com/dco). All
  contributions to this repository must be signed as described on that page.
  Your signature certifies that you wrote the patch or have the right to pass it
  on as an open-source patch.

### Getting started

The topic branch will be branched from the latest `master` and named `topic/{feature-name}`. To merge any pull request into
this branch, it will need 2 approvals from team members.

Transport Go repository, and follow the instructions in the previous link to clone
your fork and set the upstream remote to the main Transport Go repository. Because of
the DCO, set your name and e-mail in the Git configuration for signing. Finally,
create a local topic branch from the upstream `topic/{feature-name}` mentioned
above.

For instance, this setup part could look like this:

```shell
## Clone your forked repository
git clone git@github.com:<github username>/transport-go.git

## Navigate to the directory
cd transport-go

## Set name and e-mail configuration
git config user.name "John Doe"
git config user.email johndoe@example.com

## Setup the upstream remote
git remote add upstream https://github.com/vmware/transport-go.git

## Check out the upstream a topic branch for your changes
git fetch
git checkout -b topic/feature-name upstream/topic/feature-name
```

### Public API Changes

If you are making a change that changes the public API of an interface make sure
to discuss this within a proposal issue with a Transport Go team member. A proposal
allows us to plan out potential breaking changes if necessary and review the API
changes. 

### Commits

If your contribution is large, split it into smaller commits that are logically
self-contained. You can then submit them as separate pull requests that we will
review one by one and merge progressively into the topic branch.

As a rule of thumb, try to keep each pull request under a couple of hundred lines of code,
_unit tests included_. We realize this isn't always easy, and sometimes not
possible at all, so feel free to ask how to split your contribution in the
GitHub issue.

In general, it's a good idea to start coding the services first
and test them in isolation, then move to the components.

For your commit message, please use the following format:

```
<type>(optional scope): <description>
 < BLANK LINE >

[optional body]
[optional Github closing reference]

 < BLANK LINE >
Signed-off-by: Your Name <your.email@example.com>
```

`type` - could be one of `feature`, `fix`, `chore`, `docs`, `refactor`, `test`


For example, a commit message could look like this:

```
fix(bridge): add correct handling of state when performing multi-broker connections

- ensures session is created and managed properly
- pervents channel from remaining open once connection closes
- cleans up channel manager once disconnected

Close: #173

Signed-off-by: Your Name <your.email@example.com>
```

These documents provide guidance creating a well-crafted commit message:

- [How to Write a Git Commit Message](http://chris.beams.io/posts/git-commit/)
- [Closing Issues Via Commit Messages](https://help.github.com/articles/closing-issues-via-commit-messages/)
- [Conventional Commits ](https://www.conventionalcommits.org/en/v1.0.0-beta.4/)
- [Github: Closing issues using keywords](https://help.github.com/en/articles/closing-issues-using-keywords)

### Submitting pull requests

As you implement your contribution, make sure all work stays on your local topic
branch. When an isolated part of the feature is complete with unit tests, make
sure to submit your pull request **against the topic branch** on the main
Transport Go repository instead of `master`. This will allow us to accept and merge
partial changes that shouldn't make it into a production release of Transport Go yet.
We expect every pull request to come with exhaustive unit tests for the
submitted code.

**Do not, at any point, rebase your local topic branch on newer versions of `master` while your work is still in progress!**
This will create issues both for you, the reviewers, and maybe even other
developers who might submit additional commits to your topic branch if you
requested some help.

To make sure your pull request will pass our automated testing, before submitting
you should:

- Make sure `go test ./...` passes!
  
If everything passes, you can push your changes to your fork of Transport Go, and [submit a pull request](https://help.github.com/articles/about-pull-requests/).

- Assign yourself to the Pull-Request
- Assign proper labels for example if you are making documentation update only use `documentation`
- Assign connected Issue that this PR will resolve

### Taking reviews into account

During the review process of your pull request(s), some changes might be
requested by Transport Go team members. If that happens, add extra commits to your
pull request to address these changes. Make sure these new commits are also
signed and follow our commit message format.

Please keep an eye on your Pull-Request and try to address the comments, if any,
as soon as possible.

### Shipping it

Once your contribution is fully implemented, reviewed, and ready, we will rebase
the topic branch on the newest `master` and squash down to fewer commits if
needed (keeping you as the author, obviously).

```bash
$ git rebase -i master

# Rebase commits and resolve conflict, if any.

$ git push origin branch -f
```

Chances are, we will be more familiar with potential conflicts that might happen,
but we can work with you if you want to solve some conflicts yourself. Once
rebased, we will merge the topic branch into `master`, which involves a quick
internal pull request you don't have to worry about, and we will finally delete
the topic branch.

At that point, your contribution will be available in the next official release
of Transport Go

### Backport to an older version

In some cases, you will have to backport the changes into the older version.
Everything is the same here, only the target branch will be the older version
that is affected. If you are an external contributor, we will handle the
backport for you.