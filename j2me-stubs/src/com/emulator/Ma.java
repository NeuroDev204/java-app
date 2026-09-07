package com.emulator;

import java.io.InputStream;

public class Ma {
    public static InputStream getResourceAsStream(String name) {
        if (name == null) return null;
        if (!name.startsWith("/")) {
            name = "/" + name;
        }
        InputStream is = Main.class.getResourceAsStream(name);
        if (is == null) {
            is = Ma.class.getResourceAsStream(name);
        }
        return is;
    }
}
