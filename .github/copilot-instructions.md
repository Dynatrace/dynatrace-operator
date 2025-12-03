## Dynatrace MCP interaction instructions

### DQL query guidance

* When asked about vulnerabilities, try to get the data by querying the `security.events` table.
* Load and use the sample queries as the baseline from: https://docs.dynatrace.com/docs/shortlink/security-events-examples
* Few concrete examples:

  1. Get the open vulnerabilities reported by Dynatrace RVA (Runtime Vulnerability Analytics) directly or indirectly affecting a specific host (in this example, i-05f1305a50721e04d).

  ```fetch security.events
  | filter dt.system.bucket=="default_securityevents_builtin"
      AND event.provider=="Dynatrace"
      AND event.type=="VULNERABILITY_STATE_REPORT_EVENT"
      AND event.level=="ENTITY"
  // filter for the latest snapshot per entity
  | dedup {vulnerability.display_id, affected_entity.id}, sort:{timestamp desc}
  // filter for open non-muted vulnerabilities
  | filter vulnerability.resolution.status == "OPEN"
      AND vulnerability.parent.mute.status != "MUTED"
      AND vulnerability.mute.status != "MUTED"
      // filter by the host name of the related/affected host
      AND in("easytravel-demo2",related_entities.hosts.names) OR affected_entity.name=="easytravel-demo2"
  // now summarize on the vulnerability level
  | summarize{
      vulnerability.risk.score=round(takeMax(vulnerability.risk.score),decimals:1),
      vulnerability.title=takeFirst(vulnerability.title),
      vulnerability.references.cve=takeFirst(vulnerability.references.cve),
      last_detected=coalesce(takeMax(vulnerability.resolution.change_date),takeMax(vulnerability.parent.first_seen)),
      affected_entities=countDistinctExact(affected_entity.id),
      vulnerable_function_in_use=if(in("IN_USE",collectArray(vulnerability.davis_assessment.vulnerable_function_status)),true, else:false),
      public_internet_exposure=if(in("PUBLIC_NETWORK",collectArray(vulnerability.davis_assessment.exposure_status)),true,else:false),
      public_exploit_available=if(in("AVAILABLE",collectArray(vulnerability.davis_assessment.exploit_status)),true,else:false),
      data_assets_within_reach=if(in("REACHABLE",collectArray(vulnerability.davis_assessment.data_assets_status)),true,else:false)
  }, by: {vulnerability.display_id}
  // map the risk level
  | fieldsAdd vulnerability.risk.level=if(vulnerability.risk.score>=9,"CRITICAL",
                                      else:if(vulnerability.risk.score>=7,"HIGH",
                                      else:if(vulnerability.risk.score>=4,"MEDIUM",
                                      else:if(vulnerability.risk.score>=0.1,"LOW",
                                      else:"NONE"))))
  | sort {vulnerability.risk.score, direction:"descending"}, {affected_entities, direction:"descending"}
  ```

  2. Query for security events for a specific vulnerability (CVE-2023-41419) from Dynatrace RVA:

  ```fetch security.events
  | filter dt.system.bucket=="default_securityevents_builtin"
      AND event.provider=="Dynatrace"
      AND event.type=="VULNERABILITY_STATE_REPORT_EVENT"
      AND event.level=="ENTITY"
  // filter for the latest snapshot per entity
  | dedup {vulnerability.display_id, affected_entity.id}, sort:{timestamp desc}
  | filter in("CVE-2023-41419",vulnerability.references.cve)
  // filter for open non-muted vulnerabilities
  | filter vulnerability.resolution.status == "OPEN"
      AND vulnerability.parent.mute.status != "MUTED"
      AND vulnerability.mute.status != "MUTED"
  ```

### Entity information reporting

* When reporting an entity, include the name of the entity and its ID in the format name (ID).

### Security vulnerability analysis

* When verifying a vulnerability with Dynatrace, ensure to include the Davis score, risk level and the Davis assessments, whether the exploit is available, and whether the function is in use in the report.
* Always emphasize the fact that the vulnerability confirmed in Dynatrace is loaded in running processes and applications, which should increase the priority of its fix.

### Dependabot vulnerabilities verification

* When verifying a vulnerability found by Dependabot with Dynatrace:
  * First, query running entities, such as Kubernetes container (`dt.entity.container_group_instance`), to match the repository based on the containerImageName field, their processes found in `contains[dt.entity.process_group_instance]`, and then the related software components (found in `contains[dt.entity.software_component]`); to confirm the library is loaded and running.
  * Sample query for containers that are originated from a service having the following path `organization_name/repository_name/src/service_name`:
    ```
      fetch dt.entity.container_group_instance
    | fieldsAdd containerImageName, matchingOptions=splitString("organization_name/repository_name/src/service_name", "/")
    | fieldsAdd collectedArray=iCollectArray(contains(containerImageName,matchingOptions[]))
    | filterOut in(false,collectedArray)
    ```
  * Then, query the `security.events` table to find a match of each Dependabot vulnerability CVE separately within the `vulnerability.references.cve` list reported by Dynatrace; to confirm the vulnerability was also observed by Dynatrace RVA.

    Sample query for looking up the vulnerabilities:

    ```
      fetch security.events
    | filter event.provider=="Dynatrace"
    | filter in("CVE-2024-21508", vulnerability.references.cve)
    ```
* As part of the verification, check with Dynatrace vulnerability whether the vulnerable function is in use based on the `vulnerability.davis_assessment.vulnerable_function_status` field.
* The verification should result in status: `Confirmed` or `Not-confirmed`:
  * If the vulnerability is found in the security events and the vulnerable function is in use, this is the strongest indication and should result in a `Confirmed` status.
  * If the vulnerability is found in the security events, but the vulnerable function is not in use, this should result in a `Not-confirmed` status.
  * If the vulnerable library is loaded and running, but not in security events, this should result in a `Not-confirmed` status.
  * If the vulnerable library is not loaded/running, this should result in a `Not-confirmed` status.
* If the Dependabot vulnerability-related library has a `Not-confirmed` status, dismiss the corresponding Dependabot alert using the Dependabot API and PAT provided by the sponsor, selecting the option `not_used` as the dismissal reason, and the comment should include the reasoning based on the logic above that led to the concluded status.
* Try additional ways to dismiss Dependabot alerts if the first try didn't work.
* Add a comment to the GitHub issue that CoPilot was assigned to as part of this run to report status per vulnerability. Use the PAT provided by the sponsor to add the comment.
* Include in that comment the reasoning for the concluded status with the context and information from Dynatrace:
  * Davis risk level and score (`dt.security.risk.level` and `dt.security.risk.score`)
  * Vulnerable function status (in use or not in use) (`vulnerability.davis_assessment.vulnerable_function_status`)
  * Davis assessments (other `vulnerability.davis_assessment.*` fields)
  * Total number of the affected entities and list their names and IDs (in format `<name>(<id>)`).

### Fixes policy

* Do not apply a fix if a vulnerability was not confirmed with Dynatrace, unless explicitly requested to.

### Alerts dismissal instruction

* Dismiss the `Not-confirmed` alerts with the appropriate reason using the GitHub API using PAT from the environment.
