# Tech Lead Use Case

Imagine you are a tech lead, responsible for a small team, and you want to get your team bootstrapped with their own group with a few projects, one to hold your team Wiki, and another for a full-stack application that you've been working on. Within this role, you want to make sure that code quality is verified by yourself and at least one additional team member. You also want to setup your own GitLab CI runner to run your automation jobs.

This example will help you out! You can run the following commands to get bootstrapped and to see what would happen if you applied the configuration as is (and even apply it!):

```shell
terraform init
terraform plan -out plan.out
terraform apply plan.out
terraform output
```

> **NOTE**: You will want to add `-var base_uri=...` at the end, with ... replaced with the HTTP base-URI for your GitLab instance if you are not hosted on GitLab.com. You will also want to add `-var license_type=...` at the end, with ... replaced with _premium_ or _ultimate_ if you are not on a free plan.

Please read the code in this directory to further explore the implementation of this use case.
