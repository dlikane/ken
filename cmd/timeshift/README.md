20230308:

Timesheet and leave merge file attached, this will become the new input to timeshift.go

Leave value.sql produces leave hours.csv both attached.  These are the staff entitled to annual, sick, long service and compassionate leave.  Leave export defaults to 7.6 hours for a full day.  This must be replaced with the actual staff full day hours in leave hours.csv for the leave taken on that day of the week.  If Hours not =”7.6” leave unchanged.

I’m assuming you can build the sql into timeshift.go to get the output for the program to use.

All staff are entitled to time in lieu leave but this needs to be converted for the payroll system.  Staff in the leave hours.csv list have leave type “Time in Lieu” replaced with “Accrued Days”.  All other staff have leave type “Time in Lieu” replaced with “ADO”.

Sample timecard with leave and allowances below (timeshift.go already creates broken shift allowances) …

1. Use SQL instead of leave.csv file
2. Note: if on in sql response or hours are <>7.6 - leave as it is
        <Leave Type="Compassionate" Hours="3">
    For <Leave Type="Annual" Hours="7.6">
            <FromDate>2022-06-15T00:00:00</FromDate>
            <ToDate>2022-06-16T00:00:00</ToDate>
        </Leave>
    there will be entries for each day - look for day of the week in sql response and replace with corresponding hours for each weekday
    Same for: 
        <Leave Type="Sick" Hours="7.6">
        <Leave Type="Long Service" Hours="7.6">
3. if it is <Leave Type="Time in Lieu" Hours="7.6"> keep hours as it is but change to 'Accrued Days' if it in sql response and 'ADO' if not

Elizabeth Graham