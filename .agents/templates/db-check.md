# Database check: <fill: ticket>

- **Ticket:** <fill: path ของ ticket>
- **Revision:** <fill: รอบที่ n, สถานะไฟล์ดู `git-status.txt` ในโฟลเดอร์เดียวกัน>
- **Database:** <fill: ชื่อ DB ทดสอบ และ MCP server ที่ใช้ (อ่านอย่างเดียว)>

ทุกแถวมาจาก SELECT ที่รันจริง คัด query มาด้วย column ที่ spec ไม่ได้ระบุให้เขียนไว้ในส่วน "ไม่ได้ตรวจ"

| Case | Table | Column | กฎใน spec | ค่าจริง | ผล |
|---|---|---|---|---|---|
| <fill> | <fill> | <fill> | <fill> | <fill> | <fill: PASS / FAIL> |

## Query ที่รัน

```sql
<fill>
```

## ไม่ได้ตรวจ

<fill: column หรือ "ไม่มี">
