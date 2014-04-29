// Copyright (c) 2013-2014 Thorsten von Eicken
//
// EEPROM Configuration class: supports the configuration of numerous code modules via EEPROM

#ifndef EECONF_H
#define EECONF_H

// Code module IDs
#define DONOTUSE_MODULE 0
#define NET_MODULE      1
#define LOG_MODULE      2
#define NETTIME_MODULE  3
#define OWTEMP_MODULE   4
#define OWRELAY_MODULE  5
#define OWSCAN_MODULE   6

// Interface to be implemented by modules
class Configured {
public:
  uint8_t moduleId;				  // must be a unique module ID
  uint8_t configSize;				  // size of the config in bytes
  virtual void applyConfig(uint8_t *) = 0;        // callback to apply the config read from EEPROM
  virtual void receive(volatile uint8_t *pkt, uint8_t len) = 0;  // cb to process a received packet
};

// Initialize all modules based on the configuration in EEPROM
// This will read the EEPROM and call each of the modules' applyConfig callback
extern void eeconf_init(Configured **modules);

// Write an updated config to EEPROM for a specific module
extern void eeconf_write(uint8_t moduleId, void *data);

// Read the config for a specific module from EEPROM
extern bool eeconf_read(uint8_t moduleId, void *data);

// Dispatch a received packet to the appropriate Configured's receive() method.
// This is typically called after net.poll, e.g.: "if (net.poll()) config_dispatch();"
extern void eeconf_dispatch(volatile uint8_t *data=0, uint8_t len=0);

#endif // EECONF_H
