14th Feb 2022

Host: auroradb-skillsconn.cluster-ro-cpdlqlocq4io.ap-southeast-2.rds.amazonaws.com
Port: 6033
Username: skillsconn_bi
Password: see in .env

I’ve developed the attached sql to identify where worked (timesheet) hours <> Billed hours.  Output sample below.
The first Almira actually matches because her single timesheet (clock on/off) covers two shifts 2.5 + 3 = 5.5 timesheet hours.
Could you fix the sql to not report timesheets that match multiple shifts please?

SELECT 
    u.username AS 'Staff',
    t.ts_spenttime,
    b.rs_billing_quantity,
    TIME_FORMAT(t.ts_start_time, '%h:%i %p') as "Clock ON",
    TIME_FORMAT(t.ts_end_time, '%h:%i %p') as 'Clock Off',
    DATE_FORMAT(a.roster_shift_start, '%a, %d %b %Y') as 'Shift start dt',
    TIME(a.roster_shift_start) as 'Shift Start tm',
    DATE(a.roster_shift_end) as 'Shift Start dt',
    TIME(a.roster_shift_end) as 'Shift End tm',
    b.rs_billing_ratio_a,
    b.rs_billing_ratio_b,
    p.roster_position_name AS 'Participant',
    a.roster_shift_cancelled,
    a.roster_shift_note,
    t.ts_task,
    t.ts_notes,
    s.site_name
FROM
    roster_shift a,
    roster_position p,
    globe_site s,
    roster_shift_billable b,
    roster_shift_has_employee re,
    roster_employee e,
    opt_user u,

    opt_timesheet t
WHERE
    b.roster_shift_id = a.roster_shift_id
        AND p.roster_position_id = a.roster_position_id
        AND re.roster_shift_id = a.roster_shift_id
        AND re.roster_employee_id = e.roster_employee_id
        AND s.site_id = a.site_id
        AND YEAR(a.roster_shift_start) = 2022
        AND MONTH(a.roster_shift_start) = 1
        AND a.roster_shift_deleted = 'no'
        AND b.rs_billing_deleted = 'no'
        AND b.rs_billing_flag LIKE 'f%'
        AND b.rs_billing_quantity <> 0
        AND (a.roster_shift_cancelled < 10
        OR a.roster_shift_cancelled > 60)
        AND s.site_name IN ('In Home Support', 'Child and Youth', 'Our Connections')
        AND e.roster_employee_idoptuser = u.idopt_user

        AND u.idopt_user = t.idopt_user
        AND DATE(a.roster_shift_start) = t.ts_start_date
        AND DATE(a.roster_shift_end) = t.ts_end_date
        AND TIME(a.roster_shift_start) >= t.ts_start_time
        AND TIME(a.roster_shift_end) <= t.ts_end_time
        AND t.ts_status LIKE 'app%'
        AND ROUND(HOUR(t.ts_spenttime) + (MINUTE(ts_spenttime) / 60),4) <> b.rs_billing_quantity
ORDER BY u.username , t.ts_start_date
LIMIT 1000

-------------------------
-- Summary
-------------------------
SELECT 
    aa.username,
    aa.roster_date as "Billed hours", 
    tt.spenttime as "Timesheet hours", 
    aa.billing_quantity 
FROM (
    SELECT 
        u.idopt_user,
        DATE(a.roster_shift_start) roster_date,
        SUM(b.rs_billing_quantity) as billing_quantity,
        u.username,
        MIN(b.rs_billing_ratio_a),
        MIN(b.rs_billing_ratio_b),
        MIN(p.roster_position_name),
        MIN(a.roster_shift_cancelled),
        MIN(a.roster_shift_note),
        MIN(s.site_name)
    FROM
        roster_shift a,
        roster_position p,
        globe_site s,
        roster_shift_billable b,
        roster_shift_has_employee re,
        roster_employee e,
        opt_user u
    WHERE
        b.roster_shift_id = a.roster_shift_id
            AND p.roster_position_id = a.roster_position_id
            AND re.roster_shift_id = a.roster_shift_id
            AND re.roster_employee_id = e.roster_employee_id
            AND s.site_id = a.site_id
            AND a.roster_shift_deleted = 'no'
            AND b.rs_billing_deleted = 'no'
            AND b.rs_billing_flag LIKE 'f%'
            AND b.rs_billing_quantity <> 0
            AND (a.roster_shift_cancelled < 10
            OR a.roster_shift_cancelled > 60)
            AND s.site_name IN ('In Home Support', 'Child and Youth', 'Our Connections')
            AND e.roster_employee_idoptuser = u.idopt_user
            AND YEAR(a.roster_shift_start) = 2022
            AND MONTH(a.roster_shift_start) = 1
    GROUP BY u.idopt_user,
        u.username, 
        DATE(a.roster_shift_start)
) aa,
(
    SELECT 
        t.idopt_user,
        SUM(ROUND(HOUR(t.ts_spenttime) + (MINUTE(t.ts_spenttime) / 60),4)) as spenttime,
        MIN(t.ts_task),
        MIN(t.ts_notes),
        t.ts_start_date
    FROM
        opt_timesheet t
    WHERE
        t.ts_status LIKE 'app%'
        AND YEAR(t.ts_start_date) = 2022
        AND MONTH(t.ts_start_date) = 1
    GROUP BY
        t.idopt_user,
        t.ts_start_date
) tt
WHERE 
        aa.idopt_user = tt.idopt_user
        AND aa.roster_date = tt.ts_start_date
        AND tt.spenttime <> aa.billing_quantity
ORDER BY aa.username, aa.roster_date


--=========================



SELECT DISTINCT
    roster_position_name,
    roster_employee_name,
    site_name,
    roster_shift_start,
    roster_shift_end,
    a.roster_shift_id,
    roster_shift_cancelled,
    roster_shift_note
FROM
    roster_shift a,
    roster_position p,
    globe_site s,
    roster_employee e,
    roster_shift_has_employee se
WHERE
    p.roster_position_id = a.roster_position_id
        AND s.site_id = a.site_id
        AND e.roster_employee_id = se.roster_employee_id
        AND se.roster_shift_id = a.roster_shift_id
        AND site_name = 'In Home Support'
        AND YEAR(roster_shift_start) = 2021
        AND MONTH(roster_shift_start) >= 12
        AND MONTH(roster_shift_start) <= 12
        AND roster_position_enabled = 'yes'
        AND roster_employee_enabled = 'yes'
        AND roster_shift_deleted = 'no'
        AND roster_employee_deleted = 'no'
        AND roster_position_deleted = 'no'
ORDER BY roster_employee_name , roster_shift_start


SELECT DISTINCT
    p.roster_position_name,
    e.roster_employee_name,
    s.site_name,
    a1.roster_shift_start,
    a1.roster_shift_end,
    a1.roster_shift_cancelled,
    a1.roster_shift_note
FROM
    roster_shift a1,
    roster_position p,
    globe_site s,
    roster_employee e,
    roster_shift_has_employee se
WHERE
    EXISTS( SELECT
            1
        FROM
            roster_shift a2,
roster_employee e2
        WHERE
            e.roster_employee_name = e2.roster_employee_name
                AND a1.roster_shift_id <> a2.roster_shift_id
                AND (at1.roster_shift_start > a2.roster_shift_start
                AND a1.roster_shift_start < a2.roster_shift_end
                OR a1.roster_shift_end > a2.roster_shift_start
                AND a1.roster_shift_end < a2.roster_shift_end)
                AND p.roster_position_id = a1.roster_position_id
                AND s.site_id = a1.site_id
                AND e.roster_employee_id = se.roster_employee_id
                AND se.roster_shift_id = a1.roster_shift_id
                AND site_name = 'In Home Support'
                AND YEAR(roster_shift_start) = 2021
                AND MONTH(roster_shift_start) >= 12
                AND MONTH(roster_shift_start) <= 12
                AND roster_position_enabled = 'yes'
                AND roster_employee_enabled = 'yes'
                AND roster_shift_deleted = 'no'
                AND roster_employee_deleted = 'no'
                AND roster_position_deleted = 'no')
ORDER BY roster_employee_name , roster_shift_start


1. It's good you use aliases for tables, but please use them for all fields in query, otherwise I can only guess which field belongs to each table.
I guess
SELECT DISTINCT
    e.roster_employee_id,
    p.roster_position_name,
    e.roster_employee_name,
etc.

2. I don't understand why you use:
        AND MONTH(roster_shift_start) >= 12
        AND MONTH(roster_shift_start) <= 12
    isn't it the same as
        AND MONTH(roster_shift_start) = 12

3. Your origianla question. Presume you have a temporary table or query:
        emp_id, start_ts, end_ts, shift_id
    you are interested to find entry where exists start_ts or end_ts is between start_ts/end_ts for the same emp_id

        SELECT t1.emp_id, t1.shift_id
        FROM tmp t1
        WHERE EXISTS(
            SELECT 1 from tmp t2
            WHERE t1.emp_id = t2.emp_id
            AND t1.shift_id <> t2.shift_id
            AND (
                t1.start_ts > t2.start_ts AND t1.start_ts < t2.end_ts
                OR t1.end_ts > t2.start_ts AND t1.end_ts < t2.end_ts
            )
        ) 








SELECT t1.*
from 
(
SELECT
    e.roster_employee_id,
    p.roster_position_name,
    e.roster_employee_name,
    s.site_name,
    a.roster_shift_start,
    a.roster_shift_end,
    a.roster_shift_cancelled,
    a.roster_shift_note,
    a.roster_shift_id
FROM
    roster_shift a,
    roster_position p,
    globe_site s,
    roster_employee e,
    roster_shift_has_employee se
WHERE
    p.roster_position_id = a.roster_position_id
        AND s.site_id = a.site_id
        AND e.roster_employee_id = se.roster_employee_id
        AND se.roster_shift_id = a.roster_shift_id
        AND s.site_name = 'In Home Support'
        AND YEAR(a.roster_shift_start) = 2021
        AND MONTH(a.roster_shift_start) = 12
        AND p.roster_position_enabled = 'yes'
        AND roster_employee_enabled = 'yes'
        AND roster_shift_deleted = 'no'
        AND roster_employee_deleted = 'no'
        AND roster_position_deleted = 'no'
) t1
WHERE EXISTS(
    SELECT 1 from 
(
SELECT
    e.roster_employee_id,
    p.roster_position_name,
    e.roster_employee_name,
    s.site_name,
    a.roster_shift_start,
    a.roster_shift_end,
    a.roster_shift_cancelled,
    a.roster_shift_note,
    a.roster_shift_id
FROM
    roster_shift a,
    roster_position p,
    globe_site s,
    roster_employee e,
    roster_shift_has_employee se
WHERE
    p.roster_position_id = a.roster_position_id
        AND s.site_id = a.site_id
        AND e.roster_employee_id = se.roster_employee_id
        AND se.roster_shift_id = a.roster_shift_id
        AND s.site_name = 'In Home Support'
        AND YEAR(a.roster_shift_start) = 2021
        AND MONTH(a.roster_shift_start) = 12
        AND p.roster_position_enabled = 'yes'
        AND roster_employee_enabled = 'yes'
        AND roster_shift_deleted = 'no'
        AND roster_employee_deleted = 'no'
        AND roster_position_deleted = 'no'
) t2
    WHERE t1.roster_employee_id = t2.roster_employee_id
    AND t1.roster_shift_id <> t2.roster_shift_id
    AND (
        t1.roster_shift_start > t2.roster_shift_start AND t1.roster_shift_start < t2.roster_shift_end
        OR t1.roster_shift_end > t2.roster_shift_start AND t1.roster_shift_end < t2.roster_shift_end
    )
)




SELECT t1.roster_employee_name,
t1.roster_shift_start,
t1.roster_shift_end,
t1.roster_shift_cancelled,
t1.roster_shift_note,
t2.roster_shift_start,
t2.roster_shift_end,
t2.roster_shift_cancelled,
t2.roster_shift_note
from 
(
SELECT
    e.roster_employee_id,
    p.roster_position_name,
    e.roster_employee_name,
    s.site_name,
    a.roster_shift_start,
    a.roster_shift_end,
    a.roster_shift_cancelled,
    a.roster_shift_note,
    a.roster_shift_id
FROM
    roster_shift a,
    roster_position p,
    globe_site s,
    roster_employee e,
    roster_shift_has_employee se
WHERE
    p.roster_position_id = a.roster_position_id
        AND s.site_id = a.site_id
        AND e.roster_employee_id = se.roster_employee_id
        AND se.roster_shift_id = a.roster_shift_id
        AND s.site_name = 'In Home Support'
        AND YEAR(a.roster_shift_start) * 100 + MONTH(a.roster_shift_start) BETWEEN 202201 AND 202202
        AND p.roster_position_enabled = 'yes'
        AND roster_employee_enabled = 'yes'
        AND roster_shift_deleted = 'no'
        AND roster_employee_deleted = 'no'
        AND roster_position_deleted = 'no'
) t1,
(
SELECT
    e.roster_employee_id,
    p.roster_position_name,
    e.roster_employee_name,
    s.site_name,
    a.roster_shift_start,
    a.roster_shift_end,
    a.roster_shift_cancelled,
    a.roster_shift_note,
    a.roster_shift_id
FROM
    roster_shift a,
    roster_position p,
    globe_site s,
    roster_employee e,
    roster_shift_has_employee se
WHERE
    p.roster_position_id = a.roster_position_id
        AND s.site_id = a.site_id
        AND e.roster_employee_id = se.roster_employee_id
        AND se.roster_shift_id = a.roster_shift_id
        AND s.site_name = 'In Home Support'
        AND YEAR(a.roster_shift_start) * 100 + MONTH(a.roster_shift_start) BETWEEN 202201 AND 202202
        AND p.roster_position_enabled = 'yes'
        AND roster_employee_enabled = 'yes'
        AND roster_shift_deleted = 'no'
        AND roster_employee_deleted = 'no'
        AND roster_position_deleted = 'no'
) t2
WHERE t1.roster_employee_id = t2.roster_employee_id
AND t1.roster_shift_id <> t2.roster_shift_id
AND (
	t1.roster_shift_start > t2.roster_shift_start AND t1.roster_shift_start < t2.roster_shift_end
	OR t1.roster_shift_end > t2.roster_shift_start AND t1.roster_shift_end < t2.roster_shift_end
)

