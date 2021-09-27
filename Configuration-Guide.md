# Config guide / how to set up MTC.

If you don't know github actions and what they are - learn about them first here: https://docs.github.com/en/actions

1) Create an action YML - call it 'markdown.yml' or something similar and place it in the repo/.github/workflows/ folder in your repository:
```
on:
  push:
    branches: [master]
name: Markdown To Confluence
jobs:
  markdown:
    name: Markdown To Confluence Action
    runs-on: ubuntu-latest
    env: 
      PAGE-NAME: "START-HERE"
      SPACE: "XKB"
    steps:
      - name: gather branch details
        shell: bash
        run: |
             if [ -z "${GITHUB_HEAD_REF}" ]
             then
              echo NOT pull request, branch = $(echo ${GITHUB_REF#refs/heads/})
              echo "BRANCH_NAME=$(echo ${GITHUB_REF#refs/heads/})" >> $GITHUB_ENV
             else
              echo pull request, branch = $(echo ${GITHUB_HEAD_REF})
              echo "BRANCH_NAME=$(echo ${GITHUB_HEAD_REF})" >> $GITHUB_ENV
             fi
        id: extract_branch          
      - name: github auth
        shell: bash
        env:
          XIA_MACHINE_SSH_KEY: ${{ secrets.XIA_MACHINE_SSH_KEY }}
        run: |
          mkdir ~/.ssh
          echo $XIA_MACHINE_SSH_KEY | base64 --decode >> ~/.ssh/id_rsa && chmod 600 ~/.ssh/id_rsa
          ssh-keyscan github.com >> ~/.ssh/known_hosts
          git config --global url."ssh://git@github.com/".insteadOf "https://github.com/"
      - name: checkout markdown action
        uses: actions/checkout@v2-beta
        with:
          repository: xiatechs/markdown-to-confluence
          ref: refs/tags/v1.8
      - name: checkout branch
        uses: actions/checkout@v2
        with:
          ref: ${{ env.BRANCH_NAME }}
          path: "./${{ env.PAGE-NAME }}"
          fetch-depth: 0
      - name: run markdown to confluence action
        uses: ./
        with:
          key: ${{ secrets.CONFLUENCE_KEY }}
          space: "${{ env.SPACE }}"
          username: ${{ secrets.CONFLUENCE_USERNAME }}
          repo: "${{ env.PAGE-NAME }}"
          url: "https://xiatech.atlassian.net"

```

2) Edit the YML:
```
The bits you need to edit:

[from row 15]
    env: 
      PAGE-NAME: "START-HERE" #page name is the name of the page you want created in confluence
      SPACE: "XKB"            #space is the name of the space in confluence you want the page to be in
```

You can add tests/lint to the configuration if you want. 