import requests
import subprocess
import sys

# GitHub Personal Access Token
GITHUB_TOKEN = "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"

# Your GitHub username and repository name
GITHUB_USERNAME = "waodim"
GITHUB_REPO = "dynatrace-operator"


def create_branch_and_cherry_pick(_commit_hash, _destination_branch):
    try:
        # Create a new branch
        branch_name = f"cherry-pick-{_commit_hash}"
        #subprocess.run(['git', 'branch', '-D', branch_name], check=True)
        subprocess.run(['git', 'checkout', '-b', branch_name], check=True)

        # Cherry-pick the specified commit
        subprocess.run(['git', 'cherry-pick', '-m', '1', _commit_hash], check=True)

        subprocess.run(['git', 'add', '.'], check=True)
        subprocess.run(['git', 'commit'])
        # Push the new branch to GitHub
        subprocess.run(['git', 'push', 'origin', branch_name], check=True)

        # Create a new pull request
        pr_title = f"Cherry-pick {_commit_hash} into {_destination_branch}"
        pr_body = f"Same as in {_commit_hash}"
        response = requests.post(
            f"https://api.github.com/repos/{GITHUB_USERNAME}/{GITHUB_REPO}/pulls",
            headers={"Authorization": f"token {GITHUB_TOKEN}"},
            json={
                "title": pr_title,
                "head": branch_name,
                "base": _destination_branch,
                "body": pr_body
            }
        )

        pr_data = response.json()
        pr_url = pr_data["html_url"]
        print(f"Created PR: {pr_url}")

        # Merge the PR
        response = requests.put(
            f"{pr_data['url']}/merge",
            headers={"Authorization": f"token {GITHUB_TOKEN}"}
        )

        if response.status_code == 204:
            print(f"PR {pr_url} has been successfully merged.")
        else:
            print(f"Failed to merge PR: {pr_url}")

        # Delete the branch
        subprocess.run(['git', 'checkout', _destination_branch], check=True)
        subprocess.run(['git', 'branch', '-D', branch_name], check=True)

    except subprocess.CalledProcessError as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    if len(sys.argv) != 3:
        print("Usage: python cherry_pick_and_create_pr.py <commit_hash> <source_branch> <destination_branch>")
        sys.exit(1)

    commit_hash, destination_branch = sys.argv[1], sys.argv[2]
    create_branch_and_cherry_pick(commit_hash, destination_branch)
