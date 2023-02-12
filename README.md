# GittyUpSec.com

Made during [Linode+Dev](https://dev.to/toul_codes/linode-dev-hackathon-c16) hackathon

![](https://github.com/cowboyhat-io/gittyupsec.com/blob/main/public/images/gittyupsec.gif)

## Gitting Started

To use GittyUp you'll need to be an admin of the GitHub org you'd like to audit or have access to a
token with admin permissions.

We recommend that you delete the token after each report just to be safe--that includes updating the token value in GittyUp to be a blank string or gibberish

### Specific permissions for the token

- admin:org, project
- read:enterprise
- read:user, repo user:email
   
#### Why is *admin* needed?

Due to the design of the GitHub API it is required to have admin permissions to be able to read a repo's branch protections, which is one of the things GittyUp checks for.

