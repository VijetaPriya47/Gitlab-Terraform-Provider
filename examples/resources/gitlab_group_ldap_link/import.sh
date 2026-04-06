# GitLab group ldap links can be imported using an id made up of `group_id:ldap_provider:cn:filter`. CN and Filter are mutually exclusive, so one will be missing.

# If using the CN for the group link, the ID will end with a blank filter (":"). for example:
terraform import gitlab_group_ldap_link.test "12345:ldapmain:testcn:"

# If using the Filter for the group link, the ID will have two "::" in the middle due to having a blank CN. for example:
terraform import gitlab_group_ldap_link.test "12345:ldapmain::testfilter"
