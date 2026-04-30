# Notes lbh
## Notes on diff.cmd
Jag är inte säker på om vår model är korrekt. Det är svårt att hämta vaults når vi kan ha flera providers i en fil.
Problemet är att det finns x antal resource_groups och x antal vaults. Det blir nått konstigt nestlad for-loop för att hitta alla.

Kanske vi ska ha en regel som säger en policy per fil? Om man då vill spara till flera providers måste man skapa flera filer? Då vet vi att det alltid bara blir ett vault name som ska sökas genom.

Då kan man backup:a till AWS, Azure och OpenStack genom att ha tre olika filer som avser de samma resurserna.