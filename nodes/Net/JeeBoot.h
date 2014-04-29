// Copyright (c) 2013-2014 Thorsten von Eicken
// Code to pull group_id and node_id from JeeBoot at boot time

#ifndef JEEBOOT_H
#define JEEBOOT_H

// variables related to initialization
extern uint8_t jb_group_id;
extern uint8_t jb_node_id;

// reboot the jeenode and cause an upgrade check to occur. There will be single quick
// check by default, but if force==true then a full update cycle will be forced, which
// causes the jeenode to check for upgrade until a boot server responds
extern void jb_upgrade(bool force=false);

extern void jb_init3(void);

#endif /*JEEBOOT_H*/
