SELECT 
    roster_employee_name AS Staff,
    library_name AS Field,
    attribute_value AS Value
FROM
    roster_employee e,
    globe_object_has_globe_shape o,
    globe_shape s,
    globe_library l,
    globe_attribute t
WHERE
    e.roster_employee_globeuid = o.object_uid
        AND o.shape_id = s.shape_id
        AND library_shape_id = s.shape_id
        AND l.library_id = t.library_id
        AND e.roster_employee_globeuid = t.object_uid
        AND o.ohs_id = t.ohs_id
        AND roster_employee_deleted = 'no'
        AND o.ohs_deleted = 'no'
        AND s.shape_name like 'Leave%'
        and length(l.library_name) = 3
        AND attribute_islatest = 'yes'
group BY o.object_uid , s.shape_id , l.library_id, attribute_revision
LIMIT 11111