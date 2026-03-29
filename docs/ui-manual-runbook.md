# UI Manual Runbook (Post-Upgrade)

This checklist is aligned to the upgraded UI sections and is intended for quick human verification before merge/release.

## Preconditions

- App is running with valid `DCPORTAL_ADMIN_TOKEN` and `DCPORTAL_INSTALL_TOKEN`.
- At least one bot exists in admin, with one enabled install link.
- Test in both desktop and mobile viewport.

## 1) Admin Header And Navigation

- Open `/admin`.
- Verify the top hero renders `Discord Control Portal`.
- Verify `Open Install Portal` and `Logout` actions are visible and clickable.
- Click `Open Install Portal` and confirm it navigates to `/portal`.

## 2) Add Bot Card

- In `Add Bot`, verify all expected fields render: name, client ID, client secret, bot token, redirect URI, permissions, scopes.
- Confirm the write-only security hint is visible.
- Click `Open Official Calculator`:
  - with numeric client ID, ensure URL opens Discord URL generator for that app;
  - without valid numeric client ID, ensure it falls back to Discord applications page.
- Submit invalid `permissions` like `abc` and confirm request is rejected (`400` behavior).
- Submit invalid redirect URI like `javascript:alert(1)` and confirm request is rejected.

## 3) Bot Block And Secret Status

- In `Bots And Install Links`, verify each bot shows:
  - enable/disable badge,
  - `Client Secret: Configured/Missing`,
  - `Bot Token: Configured/Missing`.
- Open `Edit Bot`:
  - leave `client_secret` and `bot_token` empty, save, verify secrets are preserved;
  - set new `client_secret` and `bot_token`, save, verify governance actions still work;
  - check `Clear Bot Token`, save, verify token-dependent actions report missing token.
- Verify scopes are normalized when saved (extra spaces collapse to single spaces).

## 4) Install Links Subpanel

- Add a new install link under a bot and verify it appears immediately.
- Toggle link enabled/disabled and confirm badge + install availability behavior.
- Click `Copy Link URL` and verify clipboard contains full `/install/{link_id}` URL.
- Edit a link:
  - invalid permissions rejected,
  - invalid redirect URI rejected,
  - valid update persists.
- Delete a link and confirm it is removed from list.

## 5) Installed Guilds Table

- Verify table headers and rows render correctly.
- Test row actions:
  - `Refresh`,
  - `Revoke OAuth2`,
  - `Disconnect`,
  - `Disconnect + Blacklist`.
- Verify notices appear after each action and outcome matches behavior.
- Use `Refresh All Guild Info` and confirm notice reports count.

## 6) Guild Blacklist Table

- After a `Disconnect + Blacklist` action, confirm new row appears in blacklist section.
- Attempt reinstall for the blacklisted guild and verify callback blocks installation.

## 7) Portal Page

- Open `/portal`.
- Verify hero header, bot cards, link metadata (scopes/permissions), and install buttons.
- Confirm only enabled links for enabled bots are shown.
- If no enabled links exist, verify empty-state message is shown.

## 8) Install/Admin Login Pages

- Open `/` and `/admin/login`.
- Verify updated compact hero + login card layout.
- Submit invalid token and confirm inline error message.
- Submit valid token and confirm redirect to `/portal` or `/admin` respectively.

## 9) Callback Result Page

- Run a successful install flow and verify success card displays bot/guild metadata.
- Run a denied/error flow and verify failure title/message rendering.
- Verify `Back To Portal` action returns to `/`.

## 10) Mobile Responsiveness

- Test at <= `720px` width.
- Verify forms collapse to one column.
- Verify action buttons stack and remain tappable.
- Verify data tables remain horizontally scrollable without layout breakage.
