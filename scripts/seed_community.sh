#!/usr/bin/env bash
# Seeds real community_info rows for Next-Gen Programmers. Safe to run
# repeatedly: community_info rows upsert by slug. psql runs inside the
# compose Postgres container, so no psql install is needed on the host.
# Start the stack first:
#   cd deploy && docker compose up -d
set -euo pipefail

# docker compose resolves the compose file from the working directory.
cd "$(dirname "$0")/../deploy"

# The SQL is piped straight into the container, so the only host requirement
# is a working docker compose — no psql binary needed.
if ! docker compose version >/dev/null 2>&1; then
  echo "error: 'docker compose' is not working in this shell." >&2
  echo "check Docker Desktop is running (and WSL integration if you use WSL)." >&2
  exit 1
fi

if [ -z "$(docker compose ps --status running -q postgres)" ]; then
  echo "error: postgres container is not running." >&2
  echo "start it with: cd deploy && docker compose up -d" >&2
  exit 1
fi

# ON_ERROR_STOP makes psql exit non-zero on any SQL error, so set -e
# actually catches a failed seed instead of printing a false success.
# Bodies use $body$ dollar-quoting so apostrophes, quotes and emoji need no
# escaping at all. Safe to run repeatedly: rows upsert by slug.
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U community -d community_search <<'SQL'
INSERT INTO community_info (slug, title, body) VALUES
('about', 'About Next-Gen Programmers',
 $body$Next-Gen Programmers is an online programming community on Discord, founded in July 2025. It is built for programmers to learn, build, collaborate, and grow together, whether they are just starting out or already experienced. The community has been active on Discord since July 2025, with a focus on staying useful and contributor-driven rather than being a server people join and forget.$body$),
('mission', 'Our Mission',
 $body$Learn together. Build together. Grow together. Next-Gen Programmers exists to give programmers a place where learning, building, and collaboration happen in the same community, and where members actively contribute instead of just joining.$body$),
('what-you-can-do', 'What You Can Do Here',
 $body$Members can ask questions, share their projects, find teammates, exchange resources, and work alongside other developers. Beginners and experienced developers are both welcome.$body$),
('rules', 'Community Rules',
 $body$Community rules:
1. Respect the Community. Treat everyone with respect. Be kind and constructive. No harassment, bullying, racism, hate speech, or NSFW content.
2. No Spamming. Avoid excessive messages, pings, emojis, or irrelevant links.
3. Introduce Yourself (Mandatory). Please post an introduction in the 📝・introduction channel after joining the server.
4. Use English in Public Channels. Use English in all public channels so everyone can participate. Exceptions: the 💭・chai-tapri and 🏴‍☠️・underworld channels.
5. Self-Promotion. You may share your own projects, portfolios, blogs, or technical content where appropriate. Advertising unrelated servers, products, or services without permission is not allowed.
6. Stay Active. Members who remain inactive after joining may be removed to keep the community active. This applies to members with no messages for 20+ days after joining, or only one message within 2 months after joining.
7. Report Suspicious Behavior. If someone is creepy, inappropriate, or suspicious, report them to @MAJESTIC 🎐 or @Sir wizard with screenshots.
8. Respect Members' DMs. Do not harass, spam, advertise, or send inappropriate messages to members through DMs after meeting them in this server. Violations may result in an immediate ban.
9. No Scams or Fraud. Scamming, impersonation, phishing, fake giveaways, malicious links, or fraudulent activities will result in an immediate ban.
10. Use the Correct Channels. Keep discussions in their relevant channels. Off-topic content may be moved or removed by moderators.
11. Avoid Political & Religious Debates. Please avoid political or religious discussions that may create unnecessary conflict within the community.
12. No AI Slop. If you're sharing AI-generated content, ensure it provides value. Low-effort AI spam or mass-generated content may be removed.
13. Use Common Sense. Not every situation can be covered by the rules. Moderators may take action when necessary to maintain a healthy community.$body$),
('how-to-join', 'How to Join',
 $body$You can join the Next-Gen Programmers community on Discord using this invite link: https://discord.gg/AUz7KqDrnf . After joining, post an introduction in the introduction channel, which is mandatory, and read the community rules.$body$)
ON CONFLICT (slug) DO UPDATE
  SET title = EXCLUDED.title, body = EXCLUDED.body, updated_at = now();
SQL

echo "Seeded community info."