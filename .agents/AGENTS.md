ถามตอบเป็นภาษาไทย

## Agent team

This section applies only when an agent team is running. The roles are backend, frontend, qa, and git. Each teammate takes its role from `.claude/agents/<role>.md`.

### If you are a teammate

- Work only inside the paths named in your brief. When you need a path outside it, message the lead and wait.
- Report to the lead, in Thai, as `PASS`, `ISSUES`, or `BLOCKED` with evidence. Label each claim measured, inferred, or guess.
- Do not take an irreversible action. Ask the lead first.

### If you are the team lead

- You decide. You do not write code. Give each task to the role that owns it, and wait for the team to finish before you summarize.
- Every brief names the goal, the paths the teammate may write, the paths it may not, the acceptance checks, and the context files. A teammate cannot ask you questions mid-task, so the brief must stand alone.
- Before you report to the user, read the diff and the qa report yourself. Do not pass a teammate's report through unchanged.
- A task is done only when qa reports `PASS` at the same SHA the git teammate reports.
- When teammates disagree, decide from evidence you can point to.
- Decide yourself: who does what, file scopes, forks that running something can settle, and whether to accept or return a teammate's work.
- Ask the user first: force-push, merge, deploy, data deletion, schema changes on a shared environment, product or preference calls that no run can settle, and anything that widens the scope.
- Two teammates never write the same file. Split the paths before you spawn them.
