package com.emulator;

import java.lang.reflect.Field;
import java.lang.reflect.Method;
import java.util.ArrayList;
import java.util.List;

/**
 * AutoUpgrade: Java Runtime Hook bridging the active running J2ME JVM game session to Go.
 * Provides reflection-based inventory inspection, lucky clover / material counting,
 * and background automated equipment upgrading via packet dispatching.
 * Supports both original un-obfuscated KnightOnline jars (v402/v403) and modded x2 jars.
 */
public class AutoUpgrade {

    // Upgrade routine state variables
    private static volatile boolean isUpgrading = false;
    private static volatile Thread upgradeThread = null;

    private static volatile int activeItemId = 0;
    private static volatile int activeItemType = 0;
    private static volatile String activeItemName = "";
    private static volatile int activeCurrentLevel = 0;
    private static volatile int activeTargetLevel = 0;

    private static volatile int clover3MinLevel = 0;
    private static volatile int clover3MaxLevel = 0;
    private static volatile int clover4MinLevel = 0;
    private static volatile int clover4MaxLevel = 0;
    private static volatile boolean stopWhenOutOfClover = true;
    private static volatile int actionDelayMs = 800;

    private static volatile int totalAttempts = 0;
    private static volatile int successCount = 0;
    private static volatile int failCount = 0;
    private static volatile String lastStatusMessage = "Chua kich hoat";

    private static volatile int lastAttachedCloverId = -1;
    private static volatile int lastAttachedCloverType = -1;
    private static volatile int lastUpgradedItemId = -1;

    // ANSI Color Constants for Terminal Output
    private static final String ANSI_RESET = "\u001B[0m";
    private static final String ANSI_RED = "\u001B[31m";
    private static final String ANSI_GREEN = "\u001B[32m";
    private static final String ANSI_YELLOW = "\u001B[33m";
    private static final String ANSI_CYAN = "\u001B[36m";
    private static final String ANSI_BOLD_GREEN = "\u001B[1;32m";
    private static final String ANSI_BOLD_RED = "\u001B[1;31m";
    private static final String ANSI_BOLD_YELLOW = "\u001B[1;33m";
    private static final String ANSI_BOLD_CYAN = "\u001B[1;36m";

    public static class ItemInfo {
        public int id;
        public String name;
        public int type;
        public int level;
        public int quantity;

        public ItemInfo(int id, String name, int type, int level, int quantity) {
            this.id = id;
            this.name = name != null ? name : "";
            this.type = type;
            this.level = level;
            this.quantity = quantity;
        }
    }

    private static ClassLoader getGameClassLoader() {
        if (com.emulator.Main.j2meClassLoader != null) {
            return com.emulator.Main.j2meClassLoader;
        }
        ClassLoader cl = Thread.currentThread().getContextClassLoader();
        if (cl != null) {
            return cl;
        }
        return AutoUpgrade.class.getClassLoader();
    }

    private static String getActivePackagePrefix() {
        try {
            ClassLoader cl = getGameClassLoader();
            Class<?> x2Class = cl.loadClass("X2Midlet");
            Field instField = x2Class.getField("instance");
            Object x2Inst = instField.get(null);
            if (x2Inst != null) {
                Field curField = x2Class.getField("currentInstance");
                int cur = curField.getInt(x2Inst);
                if (cur == 2) {
                    return "x2.";
                }
            }
        } catch (Throwable ignored) {}
        return "";
    }

    private static Class<?> findGameClass(String... candidateNames) {
        ClassLoader cl = getGameClassLoader();
        String prefix = getActivePackagePrefix();
        for (String name : candidateNames) {
            if (!prefix.isEmpty()) {
                try {
                    return cl.loadClass(prefix + name);
                } catch (ClassNotFoundException ignored) {}
            }
            try {
                return cl.loadClass(name);
            } catch (ClassNotFoundException ignored) {}
        }
        return null;
    }

    private static Field findField(Class<?> clazz, String... candidateNames) {
        if (clazz == null) return null;
        for (String name : candidateNames) {
            Class<?> cur = clazz;
            while (cur != null && cur != Object.class) {
                try {
                    Field f = cur.getDeclaredField(name);
                    f.setAccessible(true);
                    return f;
                } catch (NoSuchFieldException ignored) {
                    cur = cur.getSuperclass();
                }
            }
        }
        return null;
    }

    private static Method findMethod(Class<?> clazz, String[] candidateNames, Class<?>... paramTypes) {
        if (clazz == null) return null;
        for (String name : candidateNames) {
            Class<?> cur = clazz;
            while (cur != null && cur != Object.class) {
                try {
                    Method m = cur.getDeclaredMethod(name, paramTypes);
                    m.setAccessible(true);
                    return m;
                } catch (NoSuchMethodException ignored) {
                    cur = cur.getSuperclass();
                }
            }
        }
        return null;
    }

    private static Class<?> getItemClass() {
        return findGameClass("GameObjects.Item", "bw", "x2.bw");
    }

    private static Class<?> getVectorClass() {
        return findGameClass("CLib.mVector", "et", "x2.et");
    }

    private static Class<?> getBlacksmithClass() {
        return findGameClass("InterfaceComponents.TabRebuildItem", "c", "x2.c");
    }

    private static Class<?> getCanvasClass() {
        return findGameClass("Main.GameCanvas", "fu", "x2.fu");
    }

    private static Class<?> getServiceClass() {
        return findGameClass("netcommand.global.GlobalService", "q", "x2.q");
    }

    private static Object getInventoryVector() {
        Class<?> itemClass = getItemClass();
        if (itemClass == null) return null;
        Field vField = findField(itemClass, "VecInvetoryPlayer", "V");
        if (vField == null) return null;
        try {
            return vField.get(null);
        } catch (Throwable t) {
            return null;
        }
    }

    private static int getVectorSize(Object vecObj) {
        if (vecObj == null) return 0;
        try {
            Method m = findMethod(vecObj.getClass(), new String[]{"size", "c"});
            if (m != null) {
                return ((Number) m.invoke(vecObj)).intValue();
            }
        } catch (Throwable ignored) {}
        return 0;
    }

    private static Object getVectorElementAt(Object vecObj, int index) {
        if (vecObj == null) return null;
        try {
            Method m = findMethod(vecObj.getClass(), new String[]{"elementAt", "a"}, int.class);
            if (m != null) {
                return m.invoke(vecObj, index);
            }
        } catch (Throwable ignored) {}
        return null;
    }

    private static int getItemId(Object itemObj) {
        if (itemObj == null) return 0;
        try {
            Field f = findField(itemObj.getClass(), "Id", "O");
            if (f != null) return f.getInt(itemObj);
        } catch (Throwable ignored) {}
        return 0;
    }

    private static int getItemType(Object itemObj) {
        if (itemObj == null) return 0;
        try {
            Field f = findField(itemObj.getClass(), "ItemCatagory", "u");
            if (f != null) return f.getInt(itemObj);
        } catch (Throwable ignored) {}
        return 0;
    }

    private static int getItemLevel(Object itemObj) {
        if (itemObj == null) return 0;
        try {
            Field f = findField(itemObj.getClass(), "tier", "z");
            if (f != null) {
                return f.getByte(itemObj) & 0xFF;
            }
        } catch (Throwable ignored) {}
        return 0;
    }

    private static String getItemName(Object itemObj) {
        if (itemObj == null) return "";
        try {
            Field fName = findField(itemObj.getClass(), "itemName", "g");
            if (fName != null) {
                String n = (String) fName.get(itemObj);
                if (n != null && !n.trim().isEmpty()) return n;
            }
            Field fContent = findField(itemObj.getClass(), "content", "h");
            if (fContent != null) {
                String n = (String) fContent.get(itemObj);
                if (n != null && !n.trim().isEmpty()) return n;
            }
            Field fExLv = findField(itemObj.getClass(), "itemNameExcludeLv", "i");
            if (fExLv != null) {
                String n = (String) fExLv.get(itemObj);
                if (n != null && !n.trim().isEmpty()) return n;
            }
        } catch (Throwable ignored) {}
        return "";
    }

    private static int getItemQuantity(Object itemObj) {
        if (itemObj == null) return 1;
        try {
            Field fQty = findField(itemObj.getClass(), "numPotion", "p");
            if (fQty != null) {
                int q = fQty.getInt(itemObj);
                if (q > 0) return q;
            }
            Field fTimeUse = findField(itemObj.getClass(), "timeUse");
            if (fTimeUse != null) {
                int q = fTimeUse.getInt(itemObj);
                if (q > 0) return q;
            }
        } catch (Throwable ignored) {}
        return 1;
    }

    private static int getItemTypeMaterial(Object itemObj) {
        if (itemObj == null) return 0;
        try {
            Field f = findField(itemObj.getClass(), "typeMaterial", "A");
            if (f != null) return f.getByte(itemObj) & 0xFF;
        } catch (Throwable ignored) {}
        return 0;
    }

    private static int getPlayerGold() {
        // 1. Try GameScreen.player.coin / gold (bản gốc)
        try {
            Class<?> gsClass = findGameClass("GameScreen.GameScreen");
            if (gsClass != null) {
                Field pField = findField(gsClass, "player");
                if (pField != null) {
                    Object pObj = pField.get(null);
                    if (pObj != null) {
                        Field coinF = findField(pObj.getClass(), "coin", "gold");
                        if (coinF != null) {
                            Object val = coinF.get(pObj);
                            if (val instanceof Number) {
                                return ((Number) val).intValue();
                            }
                        }
                    }
                }
            }
        } catch (Throwable ignored) {}

        // 2. Try bq.g (x2)
        try {
            Class<?> bqClass = findGameClass("bq", "x2.bq");
            if (bqClass != null) {
                Field goldField = findField(bqClass, "g");
                if (goldField != null) {
                    return goldField.getInt(null);
                }
            }
        } catch (Throwable ignored) {}
        return 0;
    }

    private static void resetServerResponseState() {
        try {
            Class<?> rebClass = getBlacksmithClass();
            if (rebClass != null) {
                Field bField = findField(rebClass, "isNextRebuild", "B");
                Field fField = findField(rebClass, "contentShow", "F");
                if (bField != null) bField.setByte(null, (byte) 0);
                if (fField != null) fField.set(null, null);
            }
        } catch (Throwable ignored) {}
    }

    private static boolean isServerResponseArrived() {
        try {
            Class<?> rebClass = getBlacksmithClass();
            if (rebClass != null) {
                Field bField = findField(rebClass, "isNextRebuild", "B");
                if (bField != null) return bField.getByte(null) == 1;
            }
        } catch (Throwable ignored) {}
        return false;
    }

    private static String getServerResponseMsg() {
        try {
            Class<?> rebClass = getBlacksmithClass();
            if (rebClass != null) {
                Field fField = findField(rebClass, "contentShow", "F");
                if (fField != null) return (String) fField.get(null);
            }
        } catch (Throwable ignored) {}
        return null;
    }

    private static void dismissPopupDialog() {
        try {
            Class<?> canClass = getCanvasClass();
            if (canClass != null) {
                Field pField = findField(canClass, "currentDialog", "p");
                if (pField != null && pField.get(null) != null) {
                    Method endM = findMethod(canClass, new String[]{"end_Dialog", "b"});
                    if (endM != null) {
                        endM.invoke(null);
                        return;
                    }
                    Object pObj = pField.get(null);
                    if (pObj != null) {
                        Method fMethod = findMethod(pObj.getClass(), new String[]{"f", "closeDialog"});
                        if (fMethod != null) fMethod.invoke(pObj);
                    }
                }
            }
        } catch (Throwable ignored) {}
    }

    /**
     * Resets lucky clover state in client memory and sends Action 1 to remove clover from blacksmith slot on server.
     */
    public static synchronized void resetCloverSlot() {
        try {
            Class<?> rebClass = getBlacksmithClass();
            int cloverIdToRemove = -1;
            int cloverTypeToRemove = -1;

            if (rebClass != null) {
                Field isLuckyField = findField(rebClass, "isLucky", "D");
                Field cloverField = findField(rebClass, "itemDaMayMan", "I");

                if (cloverField != null) {
                    Object currentClover = cloverField.get(null);
                    if (currentClover != null) {
                        cloverIdToRemove = getItemId(currentClover);
                        cloverTypeToRemove = getItemType(currentClover);
                    }
                    cloverField.set(null, null);
                }

                if (isLuckyField != null) {
                    isLuckyField.setBoolean(null, false);
                }
            }

            if (cloverIdToRemove <= 0 && lastAttachedCloverId > 0) {
                cloverIdToRemove = lastAttachedCloverId;
                cloverTypeToRemove = lastAttachedCloverType;
            }

            if (cloverIdToRemove > 0 && cloverTypeToRemove >= 0) {
                System.out.println(ANSI_CYAN + "[AutoUpgrade] Resetting clover: removing clover id=" + cloverIdToRemove + " type=" + cloverTypeToRemove + ANSI_RESET);
                sendPacketB((byte) 1, (short) cloverIdToRemove, (byte) cloverTypeToRemove);
                Thread.sleep(80);
            }
        } catch (Throwable t) {
            System.err.println("[AutoUpgrade] resetCloverSlot error: " + t.getMessage());
        } finally {
            lastAttachedCloverId = -1;
            lastAttachedCloverType = -1;
        }
    }

    private static boolean isStopCondition(String msg) {
        if (msg == null || msg.trim().isEmpty()) return false;
        String lower = msg.toLowerCase();
        // Ignore standard upgrade failure or success dialogue
        if (lower.contains("cố gắng hết sức") || lower.contains("co gang het suc") ||
            lower.contains("thành công") || lower.contains("thanh cong") ||
            lower.contains("thất bại") || lower.contains("that bai")) {
            return false;
        }
        if (lower.contains("không đủ xu") || lower.contains("khong du xu") ||
            lower.contains("không đủ vàng") || lower.contains("khong du vang") ||
            lower.contains("không đủ tiền") || lower.contains("khong du tien") ||
            lower.contains("không đủ ngọc") || lower.contains("khong du ngoc") ||
            lower.contains("không đủ đá") || lower.contains("khong du da") ||
            lower.contains("thiếu đá") || lower.contains("thieu da") ||
            lower.contains("hành trang đầy") || lower.contains("hanh trang day") ||
            lower.contains("hành trang không đủ") || lower.contains("hanh trang khong du")) {
            return true;
        }
        return false;
    }

    private static String escapeJson(String s) {
        if (s == null) return "";
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            switch (c) {
                case '"': sb.append("\\\""); break;
                case '\\': sb.append("\\\\"); break;
                case '\b': sb.append("\\b"); break;
                case '\f': sb.append("\\f"); break;
                case '\n': sb.append("\\n"); break;
                case '\r': sb.append("\\r"); break;
                case '\t': sb.append("\\t"); break;
                default:
                    if (c < ' ') {
                        String hex = Integer.toHexString(c);
                        sb.append("\\u");
                        for (int k = 0; k < 4 - hex.length(); k++) sb.append('0');
                        sb.append(hex);
                    } else {
                        sb.append(c);
                    }
                    break;
            }
        }
        return sb.toString();
    }

    private static boolean isClover3(String name) {
        if (name == null) return false;
        String lower = name.toLowerCase();
        return (lower.contains("cỏ 3") || lower.contains("co 3") ||
                lower.contains("3 lá") || lower.contains("3 la") ||
                lower.contains("cỏ ba lá") || lower.contains("co ba la") ||
                (lower.contains("cỏ") && lower.contains("3")) ||
                (lower.contains("co") && lower.contains("3")));
    }

    private static boolean isClover4(String name) {
        if (name == null) return false;
        String lower = name.toLowerCase();
        return (lower.contains("cỏ 4") || lower.contains("co 4") ||
                lower.contains("4 lá") || lower.contains("4 la") ||
                lower.contains("cỏ bốn lá") || lower.contains("co bon la") ||
                (lower.contains("cỏ") && lower.contains("4")) ||
                (lower.contains("co") && lower.contains("4")));
    }

    private static boolean isInsurance(String name) {
        if (name == null) return false;
        String lower = name.toLowerCase();
        return (lower.contains("bảo hiểm") || lower.contains("bao hiem"));
    }

    /**
     * Reads current bag items and player stats into a JSON payload.
     */
    public static synchronized String getInventoryJson() {
        int gold = 0;
        int clover3Count = 0;
        int clover4Count = 0;
        int insuranceCount = 0;
        List<ItemInfo> equipmentItems = new ArrayList<ItemInfo>();

        try {
            Object vObj = getInventoryVector();
            if (vObj != null) {
                int size = getVectorSize(vObj);
                for (int i = 0; i < size; i++) {
                    Object itemObj = getVectorElementAt(vObj, i);
                    if (itemObj == null) continue;

                    int id = getItemId(itemObj);
                    int type = getItemType(itemObj);
                    int level = getItemLevel(itemObj);
                    String name = getItemName(itemObj);
                    if (name.isEmpty()) {
                        name = "Vật phẩm #" + id;
                    }

                    int qty = getItemQuantity(itemObj);
                    int typeMat = getItemTypeMaterial(itemObj);

                    // Material / Consumable count
                    if (type == 7 || typeMat == 11 || isClover3(name) || isClover4(name) || isInsurance(name)) {
                        if (isClover4(name)) {
                            clover4Count += qty;
                        } else if (isClover3(name) || (type == 7 && typeMat == 11)) {
                            clover3Count += qty;
                        } else if (isInsurance(name)) {
                            insuranceCount += qty;
                        }
                    }

                    // Equipment items: category != 7
                    if (type != 7) {
                        equipmentItems.add(new ItemInfo(id, name, type, level, qty));
                    }
                }
            }

            gold = getPlayerGold();
        } catch (Throwable e) {
            System.err.println("[AutoUpgrade] Error reading inventory: " + e.getMessage());
            e.printStackTrace();
        }

        // Construct JSON response
        StringBuilder sb = new StringBuilder();
        sb.append("{");
        sb.append("\"gold\":").append(gold).append(",");
        sb.append("\"clover3Count\":").append(clover3Count).append(",");
        sb.append("\"clover4Count\":").append(clover4Count).append(",");
        sb.append("\"insuranceCount\":").append(insuranceCount).append(",");
        sb.append("\"items\":[");
        for (int i = 0; i < equipmentItems.size(); i++) {
            ItemInfo item = equipmentItems.get(i);
            if (i > 0) sb.append(",");
            sb.append("{");
            sb.append("\"id\":").append(item.id).append(",");
            sb.append("\"name\":\"").append(escapeJson(item.name)).append("\",");
            sb.append("\"type\":").append(item.type).append(",");
            sb.append("\"level\":").append(item.level).append(",");
            sb.append("\"quantity\":").append(item.quantity);
            sb.append("}");
        }
        sb.append("]");
        sb.append("}");

        return sb.toString();
    }

    private static Object findItemInInventory(int itemId, int itemType) {
        try {
            Object vObj = getInventoryVector();
            if (vObj == null) return null;

            int size = getVectorSize(vObj);
            for (int i = 0; i < size; i++) {
                Object item = getVectorElementAt(vObj, i);
                if (item == null) continue;
                if (getItemId(item) == itemId && getItemType(item) == itemType) {
                    return item;
                }
            }
        } catch (Throwable e) {
            System.err.println("[AutoUpgrade] findItemInInventory error: " + e.getMessage());
            e.printStackTrace();
        }
        return null;
    }

    private static Object findCloverInInventory(int cloverType) {
        // cloverType: 3 for Clover 3, 4 for Clover 4, 0 for Insurance
        try {
            Object vObj = getInventoryVector();
            if (vObj == null) return null;

            int size = getVectorSize(vObj);
            for (int i = 0; i < size; i++) {
                Object item = getVectorElementAt(vObj, i);
                if (item == null) continue;
                String name = getItemName(item);
                int qty = getItemQuantity(item);
                int type = getItemType(item);
                int typeMat = getItemTypeMaterial(item);

                if (cloverType == 3) {
                    if (isClover3(name) && qty > 0) return item;
                    if (type == 7 && typeMat == 11 && qty > 0 && !isClover4(name)) return item;
                } else if (cloverType == 4) {
                    if (isClover4(name) && qty > 0) return item;
                } else if (cloverType == 0) {
                    if (isInsurance(name) && qty > 0) return item;
                }
            }
        } catch (Throwable e) {
            System.err.println("[AutoUpgrade] findCloverInInventory error: " + e.getMessage());
            e.printStackTrace();
        }
        return null;
    }

    /**
     * Dispatches packet: Rebuild_Item(b1, s2, b3) or b(b1, s2, b3)
     */
    private static void sendPacketB(byte b1, short s2, byte b3) {
        try {
            Class<?> srvClass = getServiceClass();
            if (srvClass == null) {
                System.err.println("[AutoUpgrade] Service class not found! Network service unavailable.");
                return;
            }
            Method getInstanceMethod = findMethod(srvClass, new String[]{"gI", "a"});
            if (getInstanceMethod == null) {
                System.err.println("[AutoUpgrade] Service getInstance method not found!");
                return;
            }
            Object srvInstance = getInstanceMethod.invoke(null);
            if (srvInstance == null) {
                System.err.println("[AutoUpgrade] Service instance is null! Network service unavailable.");
                return;
            }

            Method bMethod = findMethod(srvClass, new String[]{"Rebuild_Item", "b"}, byte.class, short.class, byte.class);
            if (bMethod == null) {
                System.err.println("[AutoUpgrade] Rebuild_Item packet method not found!");
                return;
            }
            bMethod.invoke(srvInstance, b1, s2, b3);
        } catch (Throwable t) {
            System.err.println("[AutoUpgrade] Error sending packet b(" + b1 + ", " + s2 + ", " + b3 + "): " + t.getMessage());
            t.printStackTrace();
        }
    }

    /**
     * Launches background worker thread to execute automated upgrade loop.
     */
    public static synchronized void startUpgrade(
            int itemId,
            int itemType,
            int targetLevel,
            int clover3Min,
            int clover3Max,
            int clover4Min,
            int clover4Max,
            boolean stopOutOfClover,
            int delayMs
    ) {
        if (isUpgrading) {
            stopUpgrade();
        }

        // When switching equipment or starting fresh, reset clover slot immediately
        if (activeItemId != itemId || lastUpgradedItemId != itemId) {
            resetCloverSlot();
        }
        lastUpgradedItemId = itemId;

        activeItemId = itemId;
        activeItemType = itemType;
        activeTargetLevel = targetLevel;
        clover3MinLevel = clover3Min;
        clover3MaxLevel = clover3Max;
        clover4MinLevel = clover4Min;
        clover4MaxLevel = clover4Max;
        stopWhenOutOfClover = stopOutOfClover;
        actionDelayMs = Math.max(delayMs, 300);

        totalAttempts = 0;
        successCount = 0;
        failCount = 0;

        Object item = findItemInInventory(itemId, itemType);
        if (item == null) {
            lastStatusMessage = "Khong tim thay trang bi trong tui!";
            isUpgrading = false;
            return;
        }

        activeItemName = getItemName(item);
        activeCurrentLevel = getItemLevel(item);
        lastStatusMessage = "Bat dau dap " + activeItemName + " +" + activeCurrentLevel + " -> +" + activeTargetLevel;
        isUpgrading = true;

        upgradeThread = new Thread(new Runnable() {
            public void run() {
                System.out.println("[AutoUpgrade] Background upgrade worker started for item #" + activeItemId + " type=" + activeItemType + " (" + activeItemName + ") currentLv=" + activeCurrentLevel + " targetLv=" + activeTargetLevel);
                while (isUpgrading) {
                    try {
                        Object target = findItemInInventory(activeItemId, activeItemType);
                        if (target == null) {
                            lastStatusMessage = "Trang bi khong con trong tui do!";
                            isUpgrading = false;
                            break;
                        }

                        activeCurrentLevel = getItemLevel(target);
                        activeItemName = getItemName(target);

                        if (activeCurrentLevel >= activeTargetLevel) {
                            lastStatusMessage = "Chuc mung! Da dat +" + activeCurrentLevel + "!";
                            isUpgrading = false;
                            resetCloverSlot();
                            break;
                        }

                        // Determine lucky clover requirement
                        boolean needClover4 = (clover4MinLevel > 0 && clover4MaxLevel >= clover4MinLevel &&
                                               activeCurrentLevel >= clover4MinLevel && activeCurrentLevel <= clover4MaxLevel);
                        boolean needClover3 = (clover3MinLevel > 0 && clover3MaxLevel >= clover3MinLevel &&
                                               activeCurrentLevel >= clover3MinLevel && activeCurrentLevel <= clover3MaxLevel);

                        Object cloverToAttach = null;
                        if (needClover4) {
                            cloverToAttach = findCloverInInventory(4);
                            if (cloverToAttach == null && stopWhenOutOfClover) {
                                lastStatusMessage = "Het Co 4 la tai +" + activeCurrentLevel + "! Dung dap.";
                                isUpgrading = false;
                                resetCloverSlot();
                                break;
                            }
                        } else if (needClover3) {
                            cloverToAttach = findCloverInInventory(3);
                            if (cloverToAttach == null && stopWhenOutOfClover) {
                                lastStatusMessage = "Het Co 3 la tai +" + activeCurrentLevel + "! Dung dap.";
                                isUpgrading = false;
                                resetCloverSlot();
                                break;
                            }
                        } else {
                            // Current level is outside clover config range: actively discard / remove clover!
                            resetCloverSlot();
                        }

                        int prevLevel = activeCurrentLevel;
                        totalAttempts++;
                        lastStatusMessage = "Dang dap +" + activeCurrentLevel + " -> +" + (activeCurrentLevel + 1) +
                                (cloverToAttach != null ? " (co co)" : "") + "...";

                        System.out.println(ANSI_CYAN + "[AutoUpgrade] " + ANSI_YELLOW + "Attempt #" + totalAttempts + ": " + ANSI_RESET +
                                "itemId=" + activeItemId + " type=" + activeItemType + " +" + prevLevel + " -> +" + (prevLevel + 1) +
                                (cloverToAttach != null ? " (co co)" : ""));

                        // Dismiss any open popup dialog before starting attempt
                        dismissPopupDialog();

                        // Reset server response tracker
                        resetServerResponseState();

                        // Step 1: Put item into Blacksmith slot (Action 0)
                        sendPacketB((byte) 0, (short) activeItemId, (byte) activeItemType);
                        Thread.sleep(120);

                        // Step 2: Attach lucky clover if applicable (Action 0 for clover)
                        if (cloverToAttach != null) {
                            int cloverId = getItemId(cloverToAttach);
                            int cloverType = getItemType(cloverToAttach);
                            sendPacketB((byte) 0, (short) cloverId, (byte) cloverType);
                            lastAttachedCloverId = cloverId;
                            lastAttachedCloverType = cloverType;
                            Thread.sleep(120);
                        } else {
                            lastAttachedCloverId = -1;
                            lastAttachedCloverType = -1;
                        }

                        // Step 3: Select currency (Action 2: 0 = Vang/Xu)
                        sendPacketB((byte) 2, (short) 0, (byte) 0);
                        Thread.sleep(120);

                        // Step 4: Dispatch Blacksmith Upgrade Packet (Action 5)
                        sendPacketB((byte) 5, (short) activeItemId, (byte) activeItemType);

                        // Step 5: Wait for server response - poll level change and server message
                        long startWait = System.currentTimeMillis();
                        int pollDelay = Math.max(actionDelayMs, 1200);

                        while ((System.currentTimeMillis() - startWait < pollDelay) && isUpgrading) {
                            Thread.sleep(50);
                            String sMsg = getServerResponseMsg();
                            boolean respArrived = isServerResponseArrived();
                            Object itemCheck = findItemInInventory(activeItemId, activeItemType);
                            if (itemCheck != null) {
                                int checkLvl = getItemLevel(itemCheck);
                                if (checkLvl != prevLevel) {
                                    activeCurrentLevel = checkLvl;
                                    break;
                                }
                            }
                            if (respArrived || (sMsg != null && !sMsg.trim().isEmpty())) {
                                break;
                            }
                        }

                        // Inspect outcome
                        Object targetAfter = findItemInInventory(activeItemId, activeItemType);
                        if (targetAfter != null) {
                            activeCurrentLevel = getItemLevel(targetAfter);
                        }

                        String serverMsg = getServerResponseMsg();
                        if (serverMsg != null && !serverMsg.trim().isEmpty()) {
                            if (isStopCondition(serverMsg)) {
                                lastStatusMessage = serverMsg;
                                System.out.println(ANSI_BOLD_YELLOW + "[AutoUpgrade] Stopped due to server condition: " + serverMsg + ANSI_RESET);
                                isUpgrading = false;
                                resetCloverSlot();
                                break;
                            }
                        }

                        boolean hasResponse = isServerResponseArrived() || (serverMsg != null && !serverMsg.trim().isEmpty()) || (activeCurrentLevel != prevLevel);

                        if (!hasResponse) {
                            System.out.println(ANSI_YELLOW + "[AutoUpgrade] Attempt #" + totalAttempts + ": Server chua phan hoi trong thoi gian cho." + ANSI_RESET);
                            lastStatusMessage = "Dang cho server phan hoi...";
                        } else if (activeCurrentLevel > prevLevel) {
                            successCount++;
                            lastStatusMessage = "Thanh cong len +" + activeCurrentLevel + "! (" + successCount + "/" + totalAttempts + ")";
                            System.out.println(ANSI_BOLD_GREEN + "[AutoUpgrade] SUCCESS! +" + prevLevel + " -> +" + activeCurrentLevel + ANSI_RESET);
                        } else if (activeCurrentLevel < prevLevel) {
                            failCount++;
                            lastStatusMessage = "That bai tut xuong +" + activeCurrentLevel + "! (" + successCount + "/" + totalAttempts + ")";
                            System.out.println(ANSI_BOLD_RED + "[AutoUpgrade] FAILED! Downgraded to +" + activeCurrentLevel + ANSI_RESET);
                        } else {
                            failCount++;
                            if (serverMsg != null && !serverMsg.trim().isEmpty()) {
                                lastStatusMessage = serverMsg + " (+" + activeCurrentLevel + ") (" + successCount + "/" + totalAttempts + ")";
                            } else {
                                lastStatusMessage = "That bai giu nguyen +" + activeCurrentLevel + "! (" + successCount + "/" + totalAttempts + ")";
                            }
                            System.out.println(ANSI_RED + "[AutoUpgrade] FAILED! Kept +" + activeCurrentLevel + (serverMsg != null ? " [" + serverMsg + "]" : "") + ANSI_RESET);
                        }

                        // Dismiss any popup dialog so screen remains clean
                        dismissPopupDialog();

                        // Check if target reached
                        if (activeCurrentLevel >= activeTargetLevel) {
                            lastStatusMessage = "Chuc mung! Da dat +" + activeCurrentLevel + "!";
                            System.out.println(ANSI_BOLD_GREEN + "[AutoUpgrade] TARGET REACHED: +" + activeCurrentLevel + "! Congratulations!" + ANSI_RESET);
                            isUpgrading = false;
                            resetCloverSlot();
                            break;
                        }

                        // Rest delay between attempts
                        Thread.sleep(100);

                    } catch (InterruptedException ie) {
                        System.out.println(ANSI_YELLOW + "[AutoUpgrade] Background upgrade worker interrupted." + ANSI_RESET);
                        break;
                    } catch (Throwable err) {
                        System.err.println(ANSI_RED + "[AutoUpgrade] Error in upgrade loop: " + err.getMessage() + ANSI_RESET);
                        err.printStackTrace();
                        lastStatusMessage = "Loi nang cap: " + err.getMessage();
                        isUpgrading = false;
                        resetCloverSlot();
                        break;
                    }
                }
                isUpgrading = false;
                resetCloverSlot();
                System.out.println(ANSI_CYAN + "[AutoUpgrade] Background upgrade worker stopped." + ANSI_RESET);
            }
        }, "AutoUpgrade-Worker");

        upgradeThread.setDaemon(true);
        upgradeThread.start();
    }

    /**
     * Halts active upgrade routine.
     */
    public static synchronized void stopUpgrade() {
        if (isUpgrading || (upgradeThread != null && upgradeThread.isAlive())) {
            isUpgrading = false;
            if (upgradeThread != null) {
                upgradeThread.interrupt();
                upgradeThread = null;
            }
            resetCloverSlot();
            lastStatusMessage = "Da dung boi nguoi dung.";
            System.out.println("[AutoUpgrade] Upgrade halted by user.");
        }
    }

    /**
     * Returns JSON status string.
     */
    public static synchronized String getUpgradeStatusJson() {
        int clover3Count = 0;
        int clover4Count = 0;
        int insuranceCount = 0;

        // Query current material counts
        try {
            Object vObj = getInventoryVector();
            if (vObj != null) {
                int size = getVectorSize(vObj);
                for (int i = 0; i < size; i++) {
                    Object item = getVectorElementAt(vObj, i);
                    if (item == null) continue;
                    String name = getItemName(item);
                    int qty = getItemQuantity(item);
                    int type = getItemType(item);
                    int typeMat = getItemTypeMaterial(item);

                    if (isClover4(name)) {
                        clover4Count += qty;
                    } else if (isClover3(name) || (type == 7 && typeMat == 11)) {
                        clover3Count += qty;
                    } else if (isInsurance(name)) {
                        insuranceCount += qty;
                    }
                }
            }
        } catch (Throwable t) {
            // Ignore material recount failure during status poll
        }

        StringBuilder sb = new StringBuilder();
        sb.append("{");
        sb.append("\"running\":").append(isUpgrading).append(",");
        sb.append("\"itemId\":").append(activeItemId).append(",");
        sb.append("\"itemName\":\"").append(escapeJson(activeItemName)).append("\",");
        sb.append("\"itemType\":").append(activeItemType).append(",");
        sb.append("\"currentLevel\":").append(activeCurrentLevel).append(",");
        sb.append("\"targetLevel\":").append(activeTargetLevel).append(",");
        sb.append("\"clover3Count\":").append(clover3Count).append(",");
        sb.append("\"clover4Count\":").append(clover4Count).append(",");
        sb.append("\"insuranceCount\":").append(insuranceCount).append(",");
        sb.append("\"attempts\":").append(totalAttempts).append(",");
        sb.append("\"successes\":").append(successCount).append(",");
        sb.append("\"failures\":").append(failCount).append(",");
        sb.append("\"message\":\"").append(escapeJson(lastStatusMessage)).append("\",");
        sb.append("\"stopped\":").append(!isUpgrading);
        sb.append("}");

        return sb.toString();
    }

    public static void main(String[] args) {
        System.out.println("=== Testing AutoUpgrade ===");
        System.out.println("Inventory JSON: " + getInventoryJson());
        System.out.println("Status JSON: " + getUpgradeStatusJson());
        startUpgrade(100, 0, 10, 4, 8, 9, 15, true, 500);
        System.out.println("Status JSON after start: " + getUpgradeStatusJson());
        stopUpgrade();
        System.out.println("Status JSON after stop: " + getUpgradeStatusJson());
        System.out.println("=== AutoUpgrade test complete ===");
    }
}
