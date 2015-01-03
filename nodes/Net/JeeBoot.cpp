// Copyright (c) 2013-2014 Thorsten von Eicken
// Code to pull group_id and node_id from JeeBoot at boot time

#include <JeeLib.h>
#include <JeeBoot.h>
#include <avr/wdt.h>

uint8_t jb_group_id __attribute__ ((section (".noinit"))); // prevents zeroing since we init it
uint8_t jb_node_id  __attribute__ ((section (".noinit"))); // prevents zeroing since we init it

// reboot the jeenode and cause an upgrade check to occur. There will be single quick
// check by default, but if force==true then a full update cycle will be forced, which
// causes the jeenode to check for upgrade until a boot server responds
void jb_upgrade(bool force) {
  delay(10); // give serial time to print last char
  *(uint32_t*)0x100 = force ? 0x0badf00d : 0;
  wdt_enable(WDTO_15MS);
  for (;;)
    ;
}

// At boot time the JeeBoot bootloader puts the desired group_id and node_id at the
// start of RAM. We retrieve these here.
// This runs during the init3 phase http://www.nongnu.org/avr-libc/user-manual/mem_sections.html
// which is after the stack and zero-register is initialized but before .data is copied over
// and .bss is initialized
void jb_init3(void) __attribute__ ((naked)) __attribute__ ((section (".init3")));
void jb_init3(void) {
  jb_group_id = *(byte *)0x100;
  jb_node_id  = *(byte *)0x101;
}

// Calculate free memory
extern char *__bss_end; // end of statically allocated memory
extern char *__brkval;  // highest point used by dynamic allocated heap
uint16_t jb_free_ram(void) {
  uint16_t freeValue;
  if((uint16_t)__brkval == 0)
    return ((uint16_t)&freeValue) - ((uint16_t)&__bss_end);
  else
    return ((uint16_t)&freeValue) - ((uint16_t)__brkval);
}

