// Copyright (c) 2013-2014 Thorsten von Eicken
//
// EEPROM Configuration class: supports the configuration of numerous code modules via EEPROM

#include <JeeLib.h>
#include <util/crc16.h>
#include <avr/eeprom.h>
#include <EEConf.h>

#define DEBUG 0

#define EEPROM_ADDR (0x20)
#define EEPROM_MAX  (64)            // max size of a config block

static Configured  **configs = 0;   // list of modules, each implementing Configured
static uint8_t     config_cnt = 0;  // number of modules
static uint16_t    config_sz = 0;   // total size of configs in eeprom

static bool check_crc(void) {
  uint8_t *eeprom_addr = (uint8_t *)EEPROM_ADDR;
  uint16_t crc = ~0;
  for (uint16_t i=0; i<config_sz; i++)
    crc = _crc16_update(crc, eeprom_read_byte(eeprom_addr + i));
  return crc == 0;
}

static void write_crc(void) {
  uint8_t *eeprom_addr = (uint8_t *)EEPROM_ADDR;
  uint16_t crc = ~0;
  for (uint16_t i=0; i<config_sz-2; i++)
    crc = _crc16_update(crc, eeprom_read_byte(eeprom_addr + i));
  eeprom_write_word((uint16_t*)(eeprom_addr+config_sz-2), crc);
}

void eeconf_init(Configured **cf) {
  // count the number of configs
  config_cnt = 0;
  while (cf[config_cnt] != 0) config_cnt++;
  configs = cf;
  Serial.print(F("Config: "));
  Serial.print(config_cnt);
  Serial.print(F(" configs "));

  // iterate through modules to calculate total size in EEPROM
  config_sz = 0;
  for (uint8_t i=0; i<config_cnt; i++) {
    uint8_t sz = cf[i]->configSize;
    if (sz > EEPROM_MAX) {
      Serial.println();
      Serial.print(F("CONFIG: the config for module #"));
      Serial.print(i+1); Serial.print(F(" is too large ("));
      Serial.print(sz); Serial.print(F(" vs. "));
      Serial.print(EEPROM_MAX); Serial.println(F(" max)"));
      continue;
    }
    config_sz += sz;
  }
  config_sz += 2; // CRC
  Serial.print(config_sz);
  Serial.println(F(" bytes"));

  // check CRC
  if (!check_crc()) {
    // give each module's applyConfig a rain-check
    Serial.println(F("  CRC does not match!"));
    for (uint8_t i=0; i<config_cnt; i++) {
      //Serial.print(F("Raincheck for module "));
      //Serial.println(i);
      cf[i]->applyConfig(0);
    }
    return;
  }
  Serial.println(F("  CRC matches!"));

  // read each config and pass to applyConfig
  uint8_t config_block[EEPROM_MAX];
  uint8_t *eeprom_addr = (uint8_t *)EEPROM_ADDR;
  for (uint8_t i=0; i<config_cnt; i++) {
    // read from eeprom
    eeprom_read_block(config_block, eeprom_addr, cf[i]->configSize);
#if DEBUG
    Serial.print(F("  applyConfig for module "));
    Serial.print(cf[i]->moduleId);
    Serial.print(F(" 0x"));
    for (byte b=0; b<cf[i]->configSize; b++) {
      Serial.print(" ");
      Serial.print(config_block[b], HEX);
    }
    Serial.println();
#endif
    // call applyConfig
    cf[i]->applyConfig(config_block);
    eeprom_addr += cf[i]->configSize;
  }
}

void eeconf_dispatch(volatile uint8_t *data, uint8_t len) {
  // if data=0 then no args were supplied: use rf12 buffer as default
  if (data == 0) {
    data = rf12_data;
    len = rf12_len;
  }

  // we need at least one byte for module_id
  if (len < 1) return;
  uint8_t module = data[0];

  // iterate through modules and dispatch to correct one
  for (uint8_t i=0; i<config_cnt; i++) {
    uint8_t m = configs[i]->moduleId;
    if (m == module) {
      configs[i]->receive(data+1, len-1);
      return;
    }
  }
}

void eeconf_write(uint8_t module, void *data) {
  // iterate through modules and sum sizes to get offset for writing
  uint8_t *eeprom_addr = (uint8_t *)EEPROM_ADDR;
  for (uint8_t i=0; i<config_cnt; i++) {
    uint8_t m = configs[i]->moduleId;
    if (m == module) {
      // write the config block
#if DEBUG
      Serial.print(F("Config: writing EEPROM @"));
      Serial.print((long)eeprom_addr);
      Serial.print(F(" sz="));
      Serial.print(configs[i]->configSize);
      Serial.print(F(" for module "));
      Serial.println(module);
      Serial.print(F("  data is 0x"));
      for (byte b=0; b<configs[i]->configSize; b++) {
        Serial.print(" ");
        Serial.print(((uint8_t*)data)[b], HEX);
      }
      Serial.println();
#endif
      eeprom_write_block(data, eeprom_addr, configs[i]->configSize);
      // update the CRC
      write_crc();
      return;
    } else {
      eeprom_addr += configs[i]->configSize;
    }
  }
  // oops, module not found
  Serial.print(F("Config: module "));
  Serial.print(module);
  Serial.println(F(" not found in config_write"));
}

bool eeconf_read(uint8_t module, void *data) {
  // iterate through modules and sum sizes to get offset for reading
  uint8_t *eeprom_addr = (uint8_t *)EEPROM_ADDR;
  for (uint8_t i=0; i<config_cnt; i++) {
    uint8_t m = configs[i]->moduleId;
    if (m == module) {
      // read the config block
      eeprom_read_block(data, eeprom_addr, configs[i]->configSize);
#if DEBUG
      Serial.print(F("Config: reading EEPROM @"));
      Serial.print((long)eeprom_addr);
      Serial.print(F(" sz="));
      Serial.print(configs[i]->configSize);
      Serial.print(F(" for module "));
      Serial.println(module);
      Serial.print(F("  data is 0x"));
      for (byte b=0; b<configs[i]->configSize; b++) {
        Serial.print(" ");
        Serial.print(((uint8_t*)data)[b], HEX);
      }
      Serial.println();
#endif
      return true;
    } else {
      eeprom_addr += configs[i]->configSize;
    }
  }
  // oops, module not found
  Serial.print(F("Config: module "));
  Serial.print(module);
  Serial.println(F(" not found in config_read"));
  return false;
}
