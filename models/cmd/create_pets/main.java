///usr/bin/env jbang "$0" "$@" ; exit $?

//DEPS mysql:mysql-connector-java:8.0.16

import java.io.ByteArrayOutputStream;
import java.io.PrintStream;
import java.sql.Connection;
import java.sql.Date;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.util.Calendar;

import java.lang.Thread;
import java.time.format.DateTimeFormatter;
import java.time.Duration;
import java.time.ZonedDateTime;
import java.util.concurrent.TimeUnit;

class mysql_example { 
    public static void main(String[] args) {
        if (args.length<6) {
            System.out.println("usage is: mysql_example.java host port database user password iterations");
            return;
        }
        System.out.println("Starting MySQL Petstore insert example.");

        Integer iterations = Integer.parseInt(args[5]);

        ByteArrayOutputStream baos = new ByteArrayOutputStream();
        PrintStream ps = new PrintStream(baos);

        String usessl = "&useSSL=false";
        usessl = "";

        // useSSL=false is to connect to 5.7 server https://stackoverflow.com/questions/67332909/why-can-java-not-connect-to-mysql-5-7-after-the-latest-jdk-update-and-how-should
        ps.printf("jdbc:mysql://%s:%s/%s?user=%s&password=%s%s", args[0], args[1], args[2], args[3], args[4], usessl);
        String jdbcdsn = baos.toString();

        System.out.println("DSN: "+jdbcdsn);

        // Connection conn = null;
        try {
            Connection conn = DriverManager.getConnection(jdbcdsn);

            String query = "INSERT INTO pet (name, status, creation_time, update_time)"
                + " VALUES (?, ?, ?, ?)";

            Calendar calendar = Calendar.getInstance();
            Date startDate = new Date(calendar.getTime().getTime());
            Timestamp startTime = new Timestamp(calendar.getTime().getTime());

            PreparedStatement preparedStmt = conn.prepareStatement(query);

            DateTimeFormatter fmt = DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ssZZZZZ");
            ZonedDateTime dt0 = ZonedDateTime.now();
               System.out.println(fmt.format(dt0));

            for (int i = 0; i < iterations; i++) {
                preparedStmt.setString (1, "Java A" + String.valueOf(i));
                preparedStmt.setString (2, "Available");
                preparedStmt.setTimestamp (3, startTime);
                preparedStmt.setDate (4, startDate);
                preparedStmt.execute();
            }

            ZonedDateTime dt1 = ZonedDateTime.now();
            System.out.println(fmt.format(dt1));

            Duration dur = Duration.between(dt0,dt1);
            long durms = dur.toMillis();
            long durmn = TimeUnit.MILLISECONDS.toMinutes(durms)*60;
            // String time = String.format("%02d hours, %02d min, %02d sec",
            String time = String.format("%02d:%02d:%02d",
                TimeUnit.MILLISECONDS.toHours(durms),
                TimeUnit.MILLISECONDS.toMinutes(durms) - TimeUnit.MILLISECONDS.toHours(durms) * 60,
                TimeUnit.MILLISECONDS.toSeconds(durms) - durmn);
            System.out.println("time elapsed = " + time);
            System.out.println("time elapsed = "+ dur.toString());
            System.out.println("system: "+ args[0]);

            conn.close();
        } catch (SQLException ex) {
            System.out.println("SQLException: " + ex.getMessage());
            System.out.println("SQLState: " + ex.getSQLState());
            System.out.println("VendorError: " + ex.getErrorCode());
        }
    }
}