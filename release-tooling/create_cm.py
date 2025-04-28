import re
from atlassian import Jira
import subprocess
import sys
import time
import json
import os
import shutil

# Record start time for execution time tracking
start_time = time.time()

# Clear the 'releases/cpo-release/' folder if it exists
folder_path = 'releases/cpo-release/'
if os.path.exists(folder_path):
    for filename in os.listdir(folder_path):
        file_path = os.path.join(folder_path, filename)
        try:
            if os.path.isfile(file_path):
                os.remove(file_path)
            elif os.path.isdir(file_path):
                shutil.rmtree(file_path)
        except Exception as e:
            print(f"Error removing {file_path}: {e}")
else:
    print(f"Folder {folder_path} does not exist.")

# Initialize and deploy using sheepy
subprocess.run(
    ["sheepy", "init", "-t", "templates/cpo-release/meta.py", "-m", "app=true", "--output-file", "app.json"],
    check=True
)
subprocess.run(
    ["sheepy", "deploy", "-d", "releases/cpo-release/app.json", "create", "--all", "--skip-target-check"],
    check=True,
    stdin=sys.stdin,
    stdout=sys.stdout,
    stderr=sys.stderr
)

# Start 'sheepy cm' command and monitor JIRA issue extraction
command = 'echo y | sheepy cm -d releases/cpo-release/app.json create --skip-target-check'
result = subprocess.run(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)

issue_key = None

output = result.stdout.splitlines()
for line in output:
    print(line)
    match = re.search(r"CHANGE-\d+", line)
    if match:
        issue_key = match.group(0)

if issue_key:
    print(f"\nExtracted JIRA issue: {issue_key}")
else:
    print("\nNo JIRA issue found in the output.")

if result.returncode != 0:
    print(f"Warning: 'sheepy cm' command failed with exit code {result.returncode}.")

# Initialize and deploy infrastructure using sheepy
subprocess.run(
    ["sheepy", "init", "-t", "templates/cpo-release/meta.py", "-m", "infra=true", "--output-file", "infra.json"],
    check=True
)
subprocess.run(
    ["sheepy", "deploy", "-d", "releases/cpo-release/infra.json", "create", "--all"],
    check=True,
    stdin=sys.stdin,
    stdout=sys.stdout,
    stderr=sys.stderr
)

# Load and extract relevant data from the 'infra.json' file
with open('releases/cpo-release/infra.json', 'r') as file:
    data = json.load(file)

release_name = data["shepherd_releases"][0]["name"]
release_urls = [release["url"] for release in data["shepherd_releases"]]

# Load Bearer Token from file
TOKEN_FILE = "jira_pat.txt"
try:
    with open(TOKEN_FILE, "r") as file:
        BEARER_TOKEN = file.read().strip()
except FileNotFoundError:
    print(f"Error: Token file '{TOKEN_FILE}' not found!")
    exit(1)

# Jira Service Desk URL and Jira instance initialization
JIRA_URL = "https://jira-sd.mc1.oracleiaas.com"
jira = Jira(url=JIRA_URL)
jira.session.headers.update({
    "Authorization": f"Bearer {BEARER_TOKEN}",
    "Accept": "application/json",
    "Content-Type": "application/json"
})

# Fetch the existing issue and extract the deployment plan field
issue = jira.issue(issue_key)
deployment_plan_field = issue["fields"].get("customfield_10308", None)
if not deployment_plan_field:
    print("Deployment Plan field not found!")
    exit(1)

# Extract and update the deployment table with release details
header_pattern = r"\|\| Bundle \|\| Group \|\| Regions \|\| image-push \|\|"
header_match = re.search(header_pattern, deployment_plan_field)
if not header_match:
    print("Table with the specified header not found.")
    exit(1)

table_start = header_match.start()
table_end = deployment_plan_field.find("\n\n", table_start)
if table_end == -1:
    table_end = len(deployment_plan_field)

table_text = deployment_plan_field[table_start:table_end]
rows = table_text.splitlines()
header = rows[0].strip()
data_rows = rows[1:]

header += "| mapping-update "
release_index = 0

for idx, row in enumerate(data_rows):
    if row[-1] == "|":
        data_rows[idx] += f"[{release_name}|{release_urls[release_index]}] |"
        release_index += 1

updated_table = "\n".join([header] + data_rows)
updated_deployment_plan = deployment_plan_field.replace(table_text, updated_table)

# Update Jira issue with modified deployment plan
payload = {
    "fields": {
        "customfield_10308": updated_deployment_plan
    }
}
print("Updating Jira issue...")
jira.update_issue(issue_key, payload)

# Confirm the update and print execution time
elapsed_time = time.time() - start_time
print(f"Update complete.")
print(f"Created CM: {JIRA_URL}/browse/{issue_key}")
elapsed_time_minutes = int(elapsed_time // 60)
elapsed_time_seconds = int(elapsed_time % 60)
print(f"Total time taken: {elapsed_time_minutes}m{elapsed_time_seconds}s.")
